package driveapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ryangerardwilson/c3po/gdrive/internal/googleapi"
)

// doJSON sends one JSON request. body, when set, is sent as JSON. out,
// when set, receives the decoded JSON reply.
func (c *DriveClient) doJSON(method, endpoint string, params url.Values, body, out any) error {
	params.Set("alt", "json")
	params.Set("prettyPrint", "false")
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, endpoint+"?"+params.Encode(), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := googleapi.CheckResponse(resp); err != nil {
		return err
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
