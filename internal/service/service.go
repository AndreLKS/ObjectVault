package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/andrelks/objectvault/internal/metadata"
	"github.com/andrelks/objectvault/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrInvalidBucketName = errors.New("bucket name must be 3-63 characters and contain only lowercase letters, numbers, hyphens, and dots")
	ErrEmptyObjectKey    = errors.New("object key cannot be empty")
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9.-]{3,63}$`)

type BucketService interface {
	CreateBucket(ctx context.Context, name string) (*metadata.Bucket, error)
	ListBuckets(ctx context.Context) ([]metadata.Bucket, error)
}

type ObjectService interface {
	UploadObject(ctx context.Context, bucketName, key string, content io.Reader, contentType string) (*metadata.Object, error)
	DownloadObject(ctx context.Context, bucketName, key string) (*metadata.Object, io.ReadCloser, error)
	ListObjects(ctx context.Context, bucketName string) ([]metadata.Object, error)
}

type DefaultBucketService struct {
	metaStore metadata.MetadataStore
}

func NewBucketService(metaStore metadata.MetadataStore) *DefaultBucketService {
	return &DefaultBucketService{metaStore: metaStore}
}

func (s *DefaultBucketService) CreateBucket(ctx context.Context, name string) (*metadata.Bucket, error) {
	if !bucketNameRegex.MatchString(name) {
		return nil, ErrInvalidBucketName
	}
	return s.metaStore.CreateBucket(ctx, name)
}

func (s *DefaultBucketService) ListBuckets(ctx context.Context) ([]metadata.Bucket, error) {
	return s.metaStore.ListBuckets(ctx)
}

type DefaultObjectService struct {
	storageEngine storage.StorageEngine
	metaStore     metadata.MetadataStore
}

func NewObjectService(storageEngine storage.StorageEngine, metaStore metadata.MetadataStore) *DefaultObjectService {
	return &DefaultObjectService{
		storageEngine: storageEngine,
		metaStore:     metaStore,
	}
}

func (s *DefaultObjectService) UploadObject(ctx context.Context, bucketName, key string, content io.Reader, contentType string) (*metadata.Object, error) {
	if key == "" {
		return nil, ErrEmptyObjectKey
	}

	// 1. Verify bucket exists
	_, err := s.metaStore.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, err // Returns ErrBucketNotFound
	}

	// 2. Check if object already exists to get old physical file UUID (to clean up after success)
	var oldPhysicalID *uuid.UUID
	existingObj, err := s.metaStore.GetObject(ctx, bucketName, key)
	if err == nil {
		oldPhysicalID = &existingObj.PhysicalID
	}

	// 3. Generate internal physical UUID and write to StorageEngine
	newPhysicalID := uuid.New()
	sizeBytes, err := s.storageEngine.Put(newPhysicalID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to store file on disk: %w", err)
	}

	// 4. Save metadata to MetadataStore
	now := time.Now()
	obj := &metadata.Object{
		ID:          uuid.New(),
		BucketName:  bucketName,
		Key:         key,
		PhysicalID:  newPhysicalID,
		SizeBytes:   sizeBytes,
		ContentType: contentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if existingObj != nil {
		obj.ID = existingObj.ID
		obj.CreatedAt = existingObj.CreatedAt
	}

	err = s.metaStore.SaveObject(ctx, obj)
	if err != nil {
		// COMPENSATION STEP: If metadata database update fails, delete new physical file from disk
		cleanupErr := s.storageEngine.Delete(newPhysicalID)
		if cleanupErr != nil {
			return nil, fmt.Errorf("failed to save metadata: %v (compensation failed to delete payload: %v)", err, cleanupErr)
		}
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// 5. Success! Clean up old physical file if it existed
	if oldPhysicalID != nil {
		_ = s.storageEngine.Delete(*oldPhysicalID) // Best effort cleanup
	}

	return obj, nil
}

func (s *DefaultObjectService) DownloadObject(ctx context.Context, bucketName, key string) (*metadata.Object, io.ReadCloser, error) {
	if key == "" {
		return nil, nil, ErrEmptyObjectKey
	}

	// 1. Fetch metadata
	obj, err := s.metaStore.GetObject(ctx, bucketName, key)
	if err != nil {
		return nil, nil, err // Returns ErrObjectNotFound
	}

	// 2. Fetch payload from disk
	reader, err := s.storageEngine.Get(obj.PhysicalID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read payload file: %w", err)
	}

	return obj, reader, nil
}

func (s *DefaultObjectService) ListObjects(ctx context.Context, bucketName string) ([]metadata.Object, error) {
	return s.metaStore.ListObjects(ctx, bucketName)
}
