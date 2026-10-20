package transfer

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

func DownloadDirectoryAsZip(client driveapi.Client, entry driveapi.NavEntry, targetPath string) (string, error) {
	tmp, err := os.MkdirTemp("", "gdrive-dir-zip-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	localRoot := filepath.Join(tmp, entry.Name)
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		return "", err
	}
	remoteEntries, err := client.ListTree(entry.ID)
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(remoteEntries))
	for key := range remoteEntries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, relpath := range keys {
		remote := remoteEntries[relpath]
		localPath := filepath.Join(localRoot, filepath.FromSlash(remote.RelPath))
		if remote.IsDir() {
			if err := os.MkdirAll(localPath, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if _, err := client.DownloadEntry(driveapi.NavEntry{ID: remote.ID, Name: remote.Name, MimeType: remote.MimeType, ParentID: remote.ParentID}, localPath); err != nil {
			return "", err
		}
	}
	return zipDirectory(localRoot, targetPath)
}

func zipDirectory(root, targetPath string) (string, error) {
	if filepath.Ext(targetPath) != ".zip" {
		targetPath += ".zip"
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	defer writer.Close()
	return targetPath, filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(root), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			_, err := writer.Create(rel + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		part, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(part, file)
		return err
	})
}
