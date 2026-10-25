package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func (m *model) stageClip(cut bool) {
	paths := m.selectionPaths()
	if len(paths) == 0 {
		m.status = "nothing to copy"
		if cut {
			m.status = "nothing to cut"
		}
		return
	}
	m.clipSet = true
	m.clipCut = cut
	m.clipPaths = append([]string(nil), paths...)
	m.visual = false
	verb := "copied "
	if cut {
		verb = "cut "
	}
	m.status = verb + pathNames(paths)
}

func (m *model) pasteClip() {
	if !m.clipSet || len(m.clipPaths) == 0 {
		m.status = "nothing to paste"
		return
	}
	dsts := make([]string, 0, len(m.clipPaths))
	for _, src := range m.clipPaths {
		info, err := os.Lstat(src)
		if err != nil {
			m.status = "path not found"
			return
		}
		name := filepath.Base(src)
		dst := filepath.Join(m.currentDir, name)
		if samePath(src, dst) {
			m.status = name + " is already here"
			return
		}
		if _, err := os.Lstat(dst); err == nil {
			m.status = name + " exists"
			return
		}
		if info.IsDir() && dirInside(src, m.currentDir) {
			m.status = "cannot paste a directory into itself"
			return
		}
		dsts = append(dsts, dst)
	}
	label := pathNames(m.clipPaths)
	for i, src := range m.clipPaths {
		var err error
		if m.clipCut {
			err = movePath(src, dsts[i])
		} else {
			err = copyTree(src, dsts[i])
		}
		if err != nil {
			m.status = err.Error()
			m.reload()
			return
		}
	}
	last := dsts[len(dsts)-1]
	if m.clipCut {
		m.clearClip()
	}
	m.reload()
	m.selectPath(last)
	m.status = "pasted " + label
}

func (m *model) clearClip() {
	m.clipSet = false
	m.clipCut = false
	m.clipPaths = nil
}

func (m model) clipLabel() string {
	if !m.clipSet || len(m.clipPaths) == 0 {
		return ""
	}
	verb := "copy"
	if m.clipCut {
		verb = "cut"
	}
	return verb + " " + pathNames(m.clipPaths)
}

func pathNames(paths []string) string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = filepath.Base(path)
	}
	return strings.Join(names, ", ")
}

func dirInside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func movePath(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if !crossDevice(err) {
		return err
	}
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}

func crossDevice(err error) bool {
	var link *os.LinkError
	if errors.As(err, &link) {
		return errors.Is(link.Err, syscall.EXDEV)
	}
	return errors.Is(err, syscall.EXDEV)
}

func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if err := copyEntry(src, dst, info); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return nil
}

func copyEntry(src, dst string, info os.FileInfo) error {
	mode := info.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		if err := os.Mkdir(dst, mode.Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			child, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), child); err != nil {
				return err
			}
		}
		return nil
	default:
		return copyFile(src, dst, mode)
	}
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
