// Package googleapi holds the small part of the Google API client surface
// gdrive needs: the REST endpoints and the error a non-2xx response becomes.
package googleapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// DriveBase is the Drive v3 metadata endpoint.
	DriveBase = "https://www.googleapis.com/drive/v3/"
	// UploadBase is the Drive v3 media upload endpoint.
	UploadBase = "https://www.googleapis.com/upload/drive/v3/"
)

// Error is a non-2xx Google API response.
type Error struct {
	Code    int
	Message string
	Reason  string
	Body    string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("googleapi: got HTTP response code %d with body: %v", e.Code, e.Body)
	}
	text := fmt.Sprintf("googleapi: Error %d: %s", e.Code, e.Message)
	if e.Reason != "" {
		text += ", " + e.Reason
	}
	return text
}

type errorReply struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Reason string `json:"reason"`
		} `json:"errors"`
	} `json:"error"`
}

// CheckResponse returns an *Error when resp is not a 2xx response.
// It reads the body of a failed response.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	slurp, _ := io.ReadAll(resp.Body)
	apiErr := &Error{Code: resp.StatusCode, Body: string(slurp)}
	var reply errorReply
	if json.Unmarshal(slurp, &reply) == nil {
		apiErr.Message = reply.Error.Message
		if len(reply.Error.Errors) > 0 {
			apiErr.Reason = reply.Error.Errors[0].Reason
		}
	}
	return apiErr
}
