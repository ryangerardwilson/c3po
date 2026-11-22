package slackapi

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/slack/internal/anyval"
	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// requestUploadURL asks Slack where to send one file's bytes.
func (c Client) requestUploadURL(file app.UploadFile) (string, string, error) {
	info, err := os.Stat(file.Path)
	if err != nil {
		return "", "", err
	}
	data, err := c.postForm("files.getUploadURLExternal", map[string]string{
		"filename": file.Filename,
		"length":   strconv.FormatInt(info.Size(), 10),
	})
	if err != nil {
		return "", "", err
	}
	uploadURL := anyval.Str(data["upload_url"])
	fileID := anyval.Str(data["file_id"])
	if uploadURL == "" || fileID == "" {
		return "", "", fmt.Errorf("Slack did not return upload URL for %s", file.Filename)
	}
	return uploadURL, fileID, nil
}

// UploadFile uploads one file and shares it in the conversation.
func (c Client) UploadFile(channelID, threadTS string, file app.UploadFile) (string, error) {
	uploadURL, fileID, err := c.requestUploadURL(file)
	if err != nil {
		return "", err
	}
	if err := uploadRawFile(c.httpClient(), uploadURL, file.Path, file.Filename); err != nil {
		return "", err
	}
	filePayload, _ := json.Marshal([]map[string]string{{"id": fileID, "title": file.Filename}})
	payload := map[string]string{
		"channel_id": channelID,
		"files":      string(filePayload),
	}
	if threadTS != "" {
		payload["thread_ts"] = threadTS
	}
	if _, err := c.postForm("files.completeUploadExternal", payload); err != nil {
		return "", err
	}
	return fileID, nil
}

// UploadFiles uploads every file, then shares them as one message.
func (c Client) UploadFiles(channelID, threadTS, initialComment string, files []app.UploadFile) (string, error) {
	fileEntries := make([]map[string]string, 0, len(files))
	for _, file := range files {
		uploadURL, fileID, err := c.requestUploadURL(file)
		if err != nil {
			return "", err
		}
		if err := uploadRawFile(c.httpClient(), uploadURL, file.Path, file.Filename); err != nil {
			return "", err
		}
		fileEntries = append(fileEntries, map[string]string{"id": fileID, "title": file.Filename})
	}
	filesJSON, err := json.Marshal(fileEntries)
	if err != nil {
		return "", err
	}
	payload := map[string]string{
		"channel_id": channelID,
		"files":      string(filesJSON),
	}
	if initialComment != "" {
		payload["initial_comment"] = initialComment
	}
	if threadTS != "" {
		payload["thread_ts"] = threadTS
	}
	data, err := c.postForm("files.completeUploadExternal", payload)
	if err != nil {
		return "", err
	}
	return firstShareTS(data), nil
}

// firstShareTS is the ts of the first share Slack reports for the files.
func firstShareTS(data map[string]any) string {
	for _, filePayload := range anyval.List(data["files"]) {
		shares := anyval.Map(anyval.Map(filePayload)["shares"])
		for _, shareList := range shares {
			for _, rawShare := range anyval.List(shareList) {
				if ts := anyval.Str(anyval.Map(rawShare)["ts"]); ts != "" {
					return ts
				}
			}
		}
	}
	return ""
}

func uploadRawFile(httpClient *http.Client, uploadURL, path, filename string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	req, err := http.NewRequest(http.MethodPost, uploadURL, file)
	if err != nil {
		return err
	}
	if contentType := mime.TypeByExtension(filepath.Ext(filename)); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if info, err := file.Stat(); err == nil {
		req.ContentLength = info.Size()
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("file upload failed: %s %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}

// DownloadFile fetches a private file URL with the token and writes it at
// destination, creating its directory.
func (c Client) DownloadFile(downloadURL, destination string) error {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil && filepath.Dir(destination) != "." {
		return err
	}
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
