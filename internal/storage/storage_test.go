package storage

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestLocalDiskStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "objectvault-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s, err := NewLocalDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	id := uuid.New()
	content := []byte("hello storage engine")

	// 1. Put
	written, err := s.Put(id, bytes.NewReader(content))
	if err != nil {
		t.Errorf("expected no error putting file, got %v", err)
	}
	if written != int64(len(content)) {
		t.Errorf("expected written bytes to be %d, got %d", len(content), written)
	}

	// 2. Exists
	exists, err := s.Exists(id)
	if err != nil {
		t.Errorf("expected no error checking exists, got %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	// 3. Get
	reader, err := s.Get(id)
	if err != nil {
		t.Errorf("expected no error getting file, got %v", err)
	}

	readContent, err := io.ReadAll(reader)
	if err != nil {
		reader.Close()
		t.Errorf("expected no error reading content, got %v", err)
	}
	reader.Close()

	if !bytes.Equal(readContent, content) {
		t.Errorf("expected content %q, got %q", content, readContent)
	}

	// 4. Delete
	err = s.Delete(id)
	if err != nil {
		t.Errorf("expected no error deleting file, got %v", err)
	}

	// 5. Exists check after delete
	exists, err = s.Exists(id)
	if err != nil {
		t.Errorf("expected no error checking exists after delete, got %v", err)
	}
	if exists {
		t.Error("expected file to not exist after delete")
	}
}
