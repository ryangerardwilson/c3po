package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/o"
)

// driveUploader is the Drive surface the navigator uses.
type driveUploader interface {
	ListChildren(parentID string) ([]driveapi.NavEntry, error)
	FindAvailableName(parentID, name string) (string, error)
	CreateFolder(parentID, name string) (string, error)
	UploadFile(parentID, name, filePath string) (string, error)
	TrashEntry(fileID string) error
}

func pickLocal(term *terminal) (string, error) {
	start, err := os.Getwd()
	if err != nil {
		start = "."
	}
	var path string
	err = term.suspend(func() error {
		var errb bytes.Buffer
		chosen, code := o.Pick(start, &errb)
		if code != 0 {
			msg := strings.TrimSpace(errb.String())
			if msg == "" {
				msg = "could not open the file picker"
			}
			return errors.New(msg)
		}
		path = chosen
		return nil
	})
	return path, err
}

func (m *model) upload() bool {
	if m.pick == nil {
		m.status = "upload is unavailable"
		return false
	}
	path, err := m.pick()
	if err != nil {
		m.err = err
		m.status = err.Error()
		return false
	}
	if path == "" {
		m.err = nil
		m.status = "upload cancelled"
		return false
	}
	if err := uploadLocal(m.client, m.parentID, path); err != nil {
		m.err = err
		m.status = err.Error()
		return false
	}
	m.err = nil
	m.load(m.parentID)
	if m.err != nil {
		return false
	}
	m.status = "uploaded " + filepath.Base(path)
	return false
}

func uploadLocal(dest driveUploader, parentID, localPath string) error {
	info, err := localInfo(localPath)
	if err != nil {
		return err
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported local path: %s", localPath)
	}
	name, err := dest.FindAvailableName(parentID, filepath.Base(localPath))
	if err != nil {
		return err
	}
	if !info.IsDir() {
		_, err = dest.UploadFile(parentID, name, localPath)
		return err
	}
	folderID, err := dest.CreateFolder(parentID, name)
	if err != nil {
		return err
	}
	return uploadDir(dest, folderID, localPath)
}

func localInfo(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return info, err
	}
	resolved, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if resolved.IsDir() {
		return nil, fmt.Errorf("linked directory %s", path)
	}
	return resolved, nil
}

func uploadDir(dest driveUploader, parentID, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := uploadLocal(dest, parentID, path); err != nil {
			return err
		}
	}
	return nil
}
