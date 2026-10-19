package driveapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gdrive/internal/googleapi"
)

// chunkSize matches the 8 MiB chunks the Google client used. A smaller
// file goes up in one multipart request; a larger one in resumable chunks.
const chunkSize = 8 * 1024 * 1024

const maxUploadRetries = 5

// retryDelay is the base backoff between upload retries. Tests shorten it.
var retryDelay = time.Second

type uploadJob struct {
	method      string
	endpoint    string
	meta        fileMeta
	file        *os.File
	size        int64
	contentType string
}

type chunkResult struct {
	done      bool
	id        string
	next      int64
	retryable bool
}

func (c *DriveClient) upload(method, endpoint string, meta fileMeta, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	contentType, err := sniffContentType(file)
	if err != nil {
		return "", err
	}
	job := uploadJob{method: method, endpoint: endpoint, meta: meta, file: file, size: info.Size(), contentType: contentType}
	if job.size < chunkSize {
		return c.multipartUpload(job)
	}
	return c.resumableUpload(job)
}

// sniffContentType detects the media type from the first 512 bytes, the
// way the Google client did, and rewinds the file.
func sniffContentType(file *os.File) (string, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(file, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

func uploadParams(uploadType string) url.Values {
	return url.Values{
		"alt":               {"json"},
		"fields":            {"id"},
		"prettyPrint":       {"false"},
		"supportsAllDrives": {"false"},
		"uploadType":        {uploadType},
	}
}

func (c *DriveClient) multipartUpload(job uploadJob) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metaPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/json"}})
	if err != nil {
		return "", err
	}
	if err := json.NewEncoder(metaPart).Encode(job.meta); err != nil {
		return "", err
	}
	mediaPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {job.contentType}})
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(mediaPart, job.file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequest(job.method, job.endpoint+"?"+uploadParams("multipart").Encode(), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := googleapi.CheckResponse(resp); err != nil {
		return "", err
	}
	return decodeID(resp.Body)
}

func (c *DriveClient) resumableUpload(job uploadJob) (string, error) {
	location, err := c.startResumable(job)
	if err != nil {
		return "", err
	}
	offset := int64(0)
	retries := 0
	needStatus := false
	for {
		result, err := c.uploadStep(location, job, offset, needStatus)
		if err != nil && !result.retryable {
			return "", err
		}
		if result.done {
			return result.id, nil
		}
		if result.retryable {
			retries++
			if retries > maxUploadRetries {
				return "", err
			}
			time.Sleep(retryDelay * time.Duration(retries))
			needStatus = true
			continue
		}
		retries = 0
		needStatus = false
		offset = result.next
	}
}

func (c *DriveClient) startResumable(job uploadJob) (string, error) {
	data, err := json.Marshal(job.meta)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(job.method, job.endpoint+"?"+uploadParams("resumable").Encode(), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", job.contentType)
	req.Header.Set("X-Upload-Content-Length", strconv.FormatInt(job.size, 10))
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := googleapi.CheckResponse(resp); err != nil {
		return "", err
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("resumable upload returned no session location")
	}
	return location, nil
}

// uploadStep sends the chunk at offset, or, after a failure, asks the
// session how many bytes it already holds.
func (c *DriveClient) uploadStep(location string, job uploadJob, offset int64, statusOnly bool) (chunkResult, error) {
	var body io.Reader
	length := int64(0)
	contentRange := fmt.Sprintf("bytes */%d", job.size)
	if !statusOnly {
		length = min(chunkSize, job.size-offset)
		body = io.NewSectionReader(job.file, offset, length)
		contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, offset+length-1, job.size)
	}
	req, err := http.NewRequest(http.MethodPut, location, body)
	if err != nil {
		return chunkResult{}, err
	}
	req.ContentLength = length
	req.Header.Set("Content-Range", contentRange)
	resp, err := c.http.Do(req)
	if err != nil {
		return chunkResult{retryable: true}, err
	}
	defer resp.Body.Close()
	return readChunkResponse(resp)
}

func readChunkResponse(resp *http.Response) (chunkResult, error) {
	switch {
	case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
		id, err := decodeID(resp.Body)
		return chunkResult{done: err == nil, id: id}, err
	case resp.StatusCode == http.StatusPermanentRedirect:
		return chunkResult{next: nextOffset(resp.Header.Get("Range"))}, nil
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return chunkResult{retryable: true}, googleapi.CheckResponse(resp)
	default:
		return chunkResult{}, googleapi.CheckResponse(resp)
	}
}

// nextOffset reads a "bytes=0-N" Range header. No header means the
// session holds nothing yet.
func nextOffset(header string) int64 {
	_, last, ok := strings.Cut(strings.TrimPrefix(header, "bytes="), "-")
	if !ok {
		return 0
	}
	end, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0
	}
	return end + 1
}

func decodeID(body io.Reader) (string, error) {
	var result fileResource
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}
