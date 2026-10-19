// Package driveapi is the Drive v3 surface gdrive uses, on net/http.
package driveapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/auth"
	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/googleapi"
)

const FolderMime = "application/vnd.google-apps.folder"

var ExportMimeTypes = map[string]struct {
	Mime   string
	Suffix string
}{
	"application/vnd.google-apps.document":     {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx"},
	"application/vnd.google-apps.spreadsheet":  {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
	"application/vnd.google-apps.presentation": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", ".pptx"},
	"application/vnd.google-apps.drawing":      {"image/png", ".png"},
}

type RemoteEntry struct {
	ID       string
	RelPath  string
	Name     string
	ParentID string
	MimeType string
}

func (e RemoteEntry) IsDir() bool {
	return e.MimeType == FolderMime
}

type NavEntry struct {
	ID       string
	Name     string
	MimeType string
	ParentID string
}

func (e NavEntry) IsDir() bool {
	return e.MimeType == FolderMime
}

type Client interface {
	EnsureDrivePath(drivePath string) (string, error)
	ListChildren(parentID string) ([]NavEntry, error)
	GetEntry(fileID string) (NavEntry, error)
	FindChild(parentID, name, mimeType string) (*NavEntry, error)
	CreateFolder(parentID, name string) (string, error)
	FindAvailableName(parentID, name string) (string, error)
	DownloadEntry(entry NavEntry, targetPath string) (string, error)
	ListTree(rootID string) (map[string]RemoteEntry, error)
	UploadFile(parentID, name, filePath string) (string, error)
	UpdateFile(fileID, filePath string) error
	MoveEntry(fileID, newParentID, newName, oldParentID string) error
	RenameEntry(fileID, newName string) error
	DeleteEntry(fileID string) error
}

// DriveClient calls the Drive v3 REST API with an authorized HTTP client.
type DriveClient struct {
	http       *http.Client
	apiBase    string
	uploadBase string
}

type fileResource struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	MimeType string   `json:"mimeType"`
	Parents  []string `json:"parents"`
}

type fileList struct {
	NextPageToken string         `json:"nextPageToken"`
	Files         []fileResource `json:"files"`
}

type fileMeta struct {
	Name     string   `json:"name,omitempty"`
	MimeType string   `json:"mimeType,omitempty"`
	Parents  []string `json:"parents,omitempty"`
}

func New(ctx context.Context, account *config.Account) (*DriveClient, error) {
	httpClient, err := auth.HTTPClient(ctx, account)
	if err != nil {
		return nil, err
	}
	return newClient(httpClient, googleapi.DriveBase, googleapi.UploadBase), nil
}

func newClient(httpClient *http.Client, apiBase, uploadBase string) *DriveClient {
	return &DriveClient{http: httpClient, apiBase: apiBase, uploadBase: uploadBase}
}

func (c *DriveClient) EnsureDrivePath(drivePath string) (string, error) {
	parentID := "root"
	for _, segment := range strings.Split(drivePath, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		next, err := c.ensureFolder(parentID, segment)
		if err != nil {
			return "", err
		}
		parentID = next
	}
	return parentID, nil
}

func (c *DriveClient) ensureFolder(parentID, name string) (string, error) {
	existing, err := c.FindChild(parentID, name, FolderMime)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.ID, nil
	}
	return c.CreateFolder(parentID, name)
}

func (c *DriveClient) ListChildren(parentID string) ([]NavEntry, error) {
	entries := []NavEntry{}
	pageToken := ""
	for {
		params := url.Values{
			"q":                 {fmt.Sprintf("'%s' in parents and trashed = false", escapeQuery(parentID))},
			"fields":            {"nextPageToken,files(id,name,mimeType,parents)"},
			"pageSize":          {"1000"},
			"orderBy":           {"folder,name_natural"},
			"supportsAllDrives": {"false"},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}
		var response fileList
		if err := c.doJSON(http.MethodGet, c.apiBase+"files", params, nil, &response); err != nil {
			return nil, apiError(err)
		}
		for _, item := range response.Files {
			entries = append(entries, navEntry(item, parentID))
		}
		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}
	sortNavEntries(entries)
	return entries, nil
}

func sortNavEntries(entries []NavEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		left := strings.ToLower(entries[i].Name)
		right := strings.ToLower(entries[j].Name)
		if left != right {
			return left < right
		}
		return entries[i].Name < entries[j].Name
	})
}

func navEntry(item fileResource, fallbackParent string) NavEntry {
	parent := fallbackParent
	if len(item.Parents) > 0 {
		parent = item.Parents[0]
	}
	return NavEntry{ID: item.ID, Name: item.Name, MimeType: item.MimeType, ParentID: parent}
}

func (c *DriveClient) GetEntry(fileID string) (NavEntry, error) {
	params := url.Values{"fields": {"id,name,mimeType,parents"}, "supportsAllDrives": {"false"}}
	var item fileResource
	if err := c.doJSON(http.MethodGet, c.fileURL(fileID), params, nil, &item); err != nil {
		return NavEntry{}, apiError(err)
	}
	return navEntry(item, ""), nil
}

func (c *DriveClient) FindChild(parentID, name, mimeType string) (*NavEntry, error) {
	query := []string{
		fmt.Sprintf("'%s' in parents", escapeQuery(parentID)),
		fmt.Sprintf("name = '%s'", escapeQuery(name)),
		"trashed = false",
	}
	if mimeType != "" {
		query = append(query, fmt.Sprintf("mimeType = '%s'", escapeQuery(mimeType)))
	}
	params := url.Values{
		"q":                 {strings.Join(query, " and ")},
		"fields":            {"files(id,name,mimeType,parents)"},
		"pageSize":          {"10"},
		"supportsAllDrives": {"false"},
	}
	var response fileList
	if err := c.doJSON(http.MethodGet, c.apiBase+"files", params, nil, &response); err != nil {
		return nil, apiError(err)
	}
	if len(response.Files) == 0 {
		return nil, nil
	}
	entry := navEntry(response.Files[0], parentID)
	return &entry, nil
}

func (c *DriveClient) CreateFolder(parentID, name string) (string, error) {
	params := url.Values{"fields": {"id"}, "supportsAllDrives": {"false"}}
	meta := fileMeta{Name: name, MimeType: FolderMime, Parents: []string{parentID}}
	var result fileResource
	if err := c.doJSON(http.MethodPost, c.apiBase+"files", params, meta, &result); err != nil {
		return "", apiError(err)
	}
	return result.ID, nil
}

func (c *DriveClient) FindAvailableName(parentID, name string) (string, error) {
	existing, err := c.FindChild(parentID, name, "")
	if err != nil {
		return "", err
	}
	if existing == nil {
		return name, nil
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for idx := 1; idx < 10000; idx++ {
		candidate := fmt.Sprintf("%s-%d%s", base, idx, ext)
		existing, err := c.FindChild(parentID, candidate, "")
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not allocate name for %s", name)
}

func (c *DriveClient) DownloadEntry(entry NavEntry, targetPath string) (string, error) {
	targetPath = downloadTargetPath(entry, targetPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	body, err := c.openDownload(entry)
	if err != nil {
		return "", err
	}
	defer body.Close()
	if _, err := io.Copy(out, body); err != nil {
		return "", err
	}
	return targetPath, nil
}

func (c *DriveClient) openDownload(entry NavEntry) (io.ReadCloser, error) {
	if export, ok := ExportMimeTypes[entry.MimeType]; ok {
		body, err := c.download(c.fileURL(entry.ID)+"/export", url.Values{"mimeType": {export.Mime}})
		return body, apiError(err)
	}
	if strings.HasPrefix(entry.MimeType, "application/vnd.google-apps.") {
		return nil, fmt.Errorf("download not supported for Google file type `%s`", entry.MimeType)
	}
	body, err := c.download(c.fileURL(entry.ID), url.Values{"alt": {"media"}})
	return body, apiError(err)
}

func (c *DriveClient) download(endpoint string, params url.Values) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if err := googleapi.CheckResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp.Body, nil
}

type treeItem struct {
	Base     string
	ParentID string
}

func (c *DriveClient) ListTree(rootID string) (map[string]RemoteEntry, error) {
	result := map[string]RemoteEntry{}
	queue := []treeItem{{Base: "", ParentID: rootID}}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		children, err := c.ListChildren(item.ParentID)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			rel := strings.Trim(strings.TrimPrefix(item.Base+"/"+child.Name, "/"), "/")
			entry := RemoteEntry{ID: child.ID, RelPath: rel, Name: child.Name, ParentID: item.ParentID, MimeType: child.MimeType}
			result[rel] = entry
			if entry.IsDir() {
				queue = append(queue, treeItem{Base: rel, ParentID: entry.ID})
			}
		}
	}
	return result, nil
}

func (c *DriveClient) UploadFile(parentID, name, filePath string) (string, error) {
	meta := fileMeta{Name: name, Parents: []string{parentID}}
	id, err := c.upload(http.MethodPost, c.uploadBase+"files", meta, filePath)
	return id, apiError(err)
}

func (c *DriveClient) UpdateFile(fileID, filePath string) error {
	_, err := c.upload(http.MethodPatch, c.uploadBase+"files/"+url.PathEscape(fileID), fileMeta{}, filePath)
	return apiError(err)
}

func (c *DriveClient) MoveEntry(fileID, newParentID, newName, oldParentID string) error {
	params := url.Values{
		"addParents":        {newParentID},
		"removeParents":     {oldParentID},
		"fields":            {"id,parents"},
		"supportsAllDrives": {"false"},
	}
	return apiError(c.doJSON(http.MethodPatch, c.fileURL(fileID), params, fileMeta{Name: newName}, nil))
}

func (c *DriveClient) RenameEntry(fileID, newName string) error {
	params := url.Values{"fields": {"id"}, "supportsAllDrives": {"false"}}
	return apiError(c.doJSON(http.MethodPatch, c.fileURL(fileID), params, fileMeta{Name: newName}, nil))
}

func (c *DriveClient) DeleteEntry(fileID string) error {
	params := url.Values{"supportsAllDrives": {"false"}}
	return apiError(c.doJSON(http.MethodDelete, c.fileURL(fileID), params, nil, nil))
}

func (c *DriveClient) fileURL(fileID string) string {
	return c.apiBase + "files/" + url.PathEscape(fileID)
}

func downloadTargetPath(entry NavEntry, targetPath string) string {
	export, ok := ExportMimeTypes[entry.MimeType]
	if !ok {
		return targetPath
	}
	if strings.EqualFold(filepath.Ext(targetPath), export.Suffix) {
		return targetPath
	}
	ext := filepath.Ext(targetPath)
	if ext != "" {
		return strings.TrimSuffix(targetPath, ext) + export.Suffix
	}
	return targetPath + export.Suffix
}

func escapeQuery(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "'", "\\'")
}

func apiError(err error) error {
	if err == nil {
		return nil
	}
	if googleErr, ok := err.(*googleapi.Error); ok {
		if googleErr.Code == 403 && strings.Contains(googleErr.Body, "accessNotConfigured") {
			return fmt.Errorf("google drive api is disabled for this oauth project. enable Drive API in Google Cloud Console for this client id, wait a few minutes, then retry")
		}
		return fmt.Errorf("google drive api error (%d): %s", googleErr.Code, googleErr.Body)
	}
	return err
}
