package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/o"
)

// Lister lists one Drive folder.
type Lister interface {
	ListChildren(ctx context.Context, parentID string) ([]driveapi.NavEntry, error)
}

// Uploader puts a local file or folder into a Drive folder under a name
// not yet taken there.
type Uploader interface {
	FindAvailableName(ctx context.Context, parentID, name string) (string, error)
	CreateFolder(ctx context.Context, parentID, name string) (string, error)
	UploadFile(ctx context.Context, parentID, name, filePath string) (string, error)
}

// Trasher moves a Drive file to the trash.
type Trasher interface {
	TrashEntry(ctx context.Context, fileID string) error
}

// Drive is the Drive surface the navigator uses. Each call ends when its
// ctx does.
type Drive interface {
	Lister
	Uploader
	Trasher
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

// upload hands the terminal to the file picker, then uploads the choice on
// the worker. The folder's next listing replaces the cache.
func (m *model) upload() bool {
	if m.pick == nil {
		m.status = "upload is unavailable"
		return false
	}
	if m.visits > 0 {
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
	if m.client == nil {
		m.fail(errors.New("not connected to Drive"))
		return false
	}
	client, parent, name := m.client, m.parentID, filepath.Base(path)
	m.writes++
	m.err = nil
	m.status = "uploading " + name
	m.jobs.Go(func() func() {
		err := uploadWithin(client, parent, path)
		var entries []driveapi.NavEntry
		if err == nil {
			entries, err = listWithin(client, parent)
		}
		return func() {
			m.writes--
			if err != nil {
				m.fail(err)
				return
			}
			m.relist(parent, entries)
			m.err = nil
			m.status = "uploaded " + name
		}
	})
	return false
}

func uploadWithin(dest Uploader, parentID, localPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), uploadWait)
	defer cancel()
	return uploadLocal(ctx, dest, parentID, localPath)
}

func uploadLocal(ctx context.Context, dest Uploader, parentID, localPath string) error {
	info, err := localInfo(localPath)
	if err != nil {
		return err
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported local path: %s", localPath)
	}
	name, err := dest.FindAvailableName(ctx, parentID, filepath.Base(localPath))
	if err != nil {
		return err
	}
	if !info.IsDir() {
		_, err = dest.UploadFile(ctx, parentID, name, localPath)
		return err
	}
	folderID, err := dest.CreateFolder(ctx, parentID, name)
	if err != nil {
		return err
	}
	return uploadDir(ctx, dest, folderID, localPath)
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

func uploadDir(ctx context.Context, dest Uploader, parentID, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := uploadLocal(ctx, dest, parentID, path); err != nil {
			return err
		}
	}
	return nil
}
