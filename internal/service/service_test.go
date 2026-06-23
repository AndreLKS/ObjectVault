package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/andrelks/objectvault/internal/metadata"
	"github.com/google/uuid"
)

type mockStorage struct {
	putFunc    func(id uuid.UUID, content io.Reader) (int64, error)
	getFunc    func(id uuid.UUID) (io.ReadCloser, error)
	deleteFunc func(id uuid.UUID) error
	existsFunc func(id uuid.UUID) (bool, error)
}

func (m *mockStorage) Put(id uuid.UUID, content io.Reader) (int64, error) {
	return m.putFunc(id, content)
}
func (m *mockStorage) Get(id uuid.UUID) (io.ReadCloser, error) {
	return m.getFunc(id)
}
func (m *mockStorage) Delete(id uuid.UUID) error {
	return m.deleteFunc(id)
}
func (m *mockStorage) Exists(id uuid.UUID) (bool, error) {
	return m.existsFunc(id)
}

type mockMetadata struct {
	createBucketFunc func(ctx context.Context, name string) (*metadata.Bucket, error)
	getBucketFunc    func(ctx context.Context, name string) (*metadata.Bucket, error)
	listBucketsFunc  func(ctx context.Context) ([]metadata.Bucket, error)
	saveObjectFunc   func(ctx context.Context, obj *metadata.Object) error
	getObjectFunc    func(ctx context.Context, bucketName, key string) (*metadata.Object, error)
	listObjectsFunc  func(ctx context.Context, bucketName string) ([]metadata.Object, error)
}

func (m *mockMetadata) CreateBucket(ctx context.Context, name string) (*metadata.Bucket, error) {
	return m.createBucketFunc(ctx, name)
}
func (m *mockMetadata) GetBucket(ctx context.Context, name string) (*metadata.Bucket, error) {
	return m.getBucketFunc(ctx, name)
}
func (m *mockMetadata) ListBuckets(ctx context.Context) ([]metadata.Bucket, error) {
	return m.listBucketsFunc(ctx)
}
func (m *mockMetadata) SaveObject(ctx context.Context, obj *metadata.Object) error {
	return m.saveObjectFunc(ctx, obj)
}
func (m *mockMetadata) GetObject(ctx context.Context, bucketName, key string) (*metadata.Object, error) {
	return m.getObjectFunc(ctx, bucketName, key)
}
func (m *mockMetadata) ListObjects(ctx context.Context, bucketName string) ([]metadata.Object, error) {
	return m.listObjectsFunc(ctx, bucketName)
}

func TestBucketValidation(t *testing.T) {
	meta := &mockMetadata{
		createBucketFunc: func(ctx context.Context, name string) (*metadata.Bucket, error) {
			return &metadata.Bucket{ID: 1, Name: name, CreatedAt: time.Now()}, nil
		},
	}
	svc := NewBucketService(meta)

	tests := []struct {
		name    string
		errExpr error
	}{
		{"my-bucket", nil},
		{"my.bucket", nil},
		{"my-bucket-123", nil},
		{"ab", ErrInvalidBucketName}, // too short
		{repeatString("a", 64), ErrInvalidBucketName}, // too long
		{"MY-BUCKET", ErrInvalidBucketName}, // uppercase
		{"my_bucket", ErrInvalidBucketName}, // underscores not allowed in s3 strict mode (only lowercase alphanumeric, hyphens, dots)
	}

	for _, tc := range tests {
		_, err := svc.CreateBucket(context.Background(), tc.name)
		if !errors.Is(err, tc.errExpr) {
			t.Errorf("bucket name %q: expected error %v, got %v", tc.name, tc.errExpr, err)
		}
	}
}

// Custom helper because string.Repeat is strings.Repeat
func repeatString(s string, count int) string {
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// We wrapper tc.name repetition using strings or custom helper
type repeatStr struct{}
func (repeatStr) Repeat(s string, count int) string {
	return repeatString(s, count)
}

func TestUploadObjectCompensation(t *testing.T) {
	// Assert variables
	var putCalled, deleteCalled, getObjectCalled, saveObjectCalled bool
	var deletedUUID uuid.UUID

	store := &mockStorage{
		putFunc: func(id uuid.UUID, content io.Reader) (int64, error) {
			putCalled = true
			return 12, nil
		},
		deleteFunc: func(id uuid.UUID) error {
			deleteCalled = true
			deletedUUID = id
			return nil
		},
	}

	meta := &mockMetadata{
		getBucketFunc: func(ctx context.Context, name string) (*metadata.Bucket, error) {
			return &metadata.Bucket{ID: 1, Name: name}, nil
		},
		getObjectFunc: func(ctx context.Context, bucketName, key string) (*metadata.Object, error) {
			getObjectCalled = true
			return nil, metadata.ErrObjectNotFound
		},
		saveObjectFunc: func(ctx context.Context, obj *metadata.Object) error {
			saveObjectCalled = true
			// Force DB failure
			return errors.New("db disconnect error")
		},
	}

	svc := NewObjectService(store, meta)

	_, err := svc.UploadObject(context.Background(), "my-bucket", "test-key", bytes.NewReader([]byte("test content")), "text/plain")

	if err == nil {
		t.Fatal("expected upload error, got nil")
	}

	if !getObjectCalled {
		t.Error("expected MetadataStore.GetObject to be called to check for existing object")
	}
	if !putCalled {
		t.Error("expected Storage.Put to be called")
	}
	if !saveObjectCalled {
		t.Error("expected MetadataStore.SaveObject to be called")
	}
	if !deleteCalled {
		t.Error("expected Storage.Delete (compensation) to be called after db failure")
	}
	if deletedUUID == uuid.Nil {
		t.Error("expected non-nil uuid to be deleted during compensation")
	}
}
