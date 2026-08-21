package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Store interface {
	Put(key string, src io.Reader) error
	Open(key string) (ReadSeekCloser, error)
	Path(key string) string
	Remove(key string) error
}

type LocalStore struct{ root string }

func NewLocal(root string) (*LocalStore, error) {
	if root == "" {
		return nil, fmt.Errorf("storage root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &LocalStore{root: root}, nil
}

func (s *LocalStore) resolve(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.root, clean), nil
}

func (s *LocalStore) Put(key string, src io.Reader) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = io.Copy(tmp, src); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *LocalStore) Open(key string) (ReadSeekCloser, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}
func (s *LocalStore) Path(key string) string {
	path, err := s.resolve(key)
	if err != nil {
		return ""
	}
	return path
}
func (s *LocalStore) Remove(key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
