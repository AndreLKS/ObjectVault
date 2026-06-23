package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// StorageEngine defines raw I/O operations for object payloads.
type StorageEngine interface {
	Put(id uuid.UUID, content io.Reader) (int64, error)
	Get(id uuid.UUID) (io.ReadCloser, error)
	Delete(id uuid.UUID) error
	Exists(id uuid.UUID) (bool, error)
}

// LocalDiskStorage implements StorageEngine on the local file system.
type LocalDiskStorage struct {
	baseDir string
}

// NewLocalDiskStorage creates a new LocalDiskStorage engine, ensuring baseDir exists.
func NewLocalDiskStorage(baseDir string) (*LocalDiskStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %q: %w", baseDir, err)
	}
	return &LocalDiskStorage{baseDir: baseDir}, nil
}

func (s *LocalDiskStorage) getFilePath(id uuid.UUID) string {
	return filepath.Join(s.baseDir, id.String())
}

// Put writes the contents of the reader to a file named after the UUID.
func (s *LocalDiskStorage) Put(id uuid.UUID, content io.Reader) (int64, error) {
	filePath := s.getFilePath(id)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create storage file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, content)
	if err != nil {
		// Clean up the partial file if write fails
		os.Remove(filePath)
		return 0, fmt.Errorf("failed to write payload: %w", err)
	}

	return written, nil
}

// Get opens the storage file for reading.
func (s *LocalDiskStorage) Get(id uuid.UUID) (io.ReadCloser, error) {
	filePath := s.getFilePath(id)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object file %s not found: %w", id.String(), os.ErrNotExist)
		}
		return nil, fmt.Errorf("failed to open object file: %w", err)
	}
	return file, nil
}

// Delete removes the storage file from disk.
func (s *LocalDiskStorage) Delete(id uuid.UUID) error {
	filePath := s.getFilePath(id)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted or doesn't exist, which is fine for idempotency
		}
		return fmt.Errorf("failed to delete object file: %w", err)
	}
	return nil
}

// Exists checks if the storage file exists on disk.
func (s *LocalDiskStorage) Exists(id uuid.UUID) (bool, error) {
	filePath := s.getFilePath(id)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file: %w", err)
	}
	return !info.IsDir(), nil
}
