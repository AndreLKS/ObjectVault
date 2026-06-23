package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrObjectNotFound = errors.New("object not found")
	ErrBucketExists   = errors.New("bucket already exists")
)

type Bucket struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Object struct {
	ID          uuid.UUID `json:"id"`
	BucketName  string    `json:"bucket_name"`
	Key         string    `json:"key"`
	PhysicalID  uuid.UUID `json:"physical_id"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MetadataStore interface {
	CreateBucket(ctx context.Context, name string) (*Bucket, error)
	GetBucket(ctx context.Context, name string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]Bucket, error)

	SaveObject(ctx context.Context, obj *Object) error
	GetObject(ctx context.Context, bucketName, key string) (*Object, error)
	ListObjects(ctx context.Context, bucketName string) ([]Object, error)
}

type PostgresMetadataStore struct {
	db *sql.DB
}

func NewPostgresMetadataStore(db *sql.DB) *PostgresMetadataStore {
	return &PostgresMetadataStore{db: db}
}

func (s *PostgresMetadataStore) CreateBucket(ctx context.Context, name string) (*Bucket, error) {
	var b Bucket
	query := `INSERT INTO buckets (name) VALUES ($1) RETURNING id, name, created_at`
	err := s.db.QueryRowContext(ctx, query, name).Scan(&b.ID, &b.Name, &b.CreatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrBucketExists
		}
		return nil, fmt.Errorf("failed to create bucket in db: %w", err)
	}
	return &b, nil
}

func (s *PostgresMetadataStore) GetBucket(ctx context.Context, name string) (*Bucket, error) {
	var b Bucket
	query := `SELECT id, name, created_at FROM buckets WHERE name = $1`
	err := s.db.QueryRowContext(ctx, query, name).Scan(&b.ID, &b.Name, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBucketNotFound
		}
		return nil, fmt.Errorf("failed to get bucket from db: %w", err)
	}
	return &b, nil
}

func (s *PostgresMetadataStore) ListBuckets(ctx context.Context) ([]Bucket, error) {
	query := `SELECT id, name, created_at FROM buckets ORDER BY name ASC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query buckets: %w", err)
	}
	defer rows.Close()

	var buckets []Bucket
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bucket row: %w", err)
		}
		buckets = append(buckets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bucket rows iteration error: %w", err)
	}

	return buckets, nil
}

func (s *PostgresMetadataStore) SaveObject(ctx context.Context, obj *Object) error {
	query := `
		INSERT INTO objects (id, bucket_name, key, physical_id, size_bytes, content_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (bucket_name, key) DO UPDATE SET
			id = EXCLUDED.id,
			physical_id = EXCLUDED.physical_id,
			size_bytes = EXCLUDED.size_bytes,
			content_type = EXCLUDED.content_type,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		obj.ID,
		obj.BucketName,
		obj.Key,
		obj.PhysicalID,
		obj.SizeBytes,
		obj.ContentType,
		obj.CreatedAt,
		obj.UpdatedAt,
	)
	if err != nil {
		// If reference constraint on bucket_name fails
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return ErrBucketNotFound
		}
		return fmt.Errorf("failed to save object to db: %w", err)
	}
	return nil
}

func (s *PostgresMetadataStore) GetObject(ctx context.Context, bucketName, key string) (*Object, error) {
	var obj Object
	query := `
		SELECT id, bucket_name, key, physical_id, size_bytes, content_type, created_at, updated_at
		FROM objects
		WHERE bucket_name = $1 AND key = $2
	`
	err := s.db.QueryRowContext(ctx, query, bucketName, key).Scan(
		&obj.ID,
		&obj.BucketName,
		&obj.Key,
		&obj.PhysicalID,
		&obj.SizeBytes,
		&obj.ContentType,
		&obj.CreatedAt,
		&obj.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to get object from db: %w", err)
	}
	return &obj, nil
}

func (s *PostgresMetadataStore) ListObjects(ctx context.Context, bucketName string) ([]Object, error) {
	// First check if bucket exists so we can return ErrBucketNotFound if appropriate
	_, err := s.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, bucket_name, key, physical_id, size_bytes, content_type, created_at, updated_at
		FROM objects
		WHERE bucket_name = $1
		ORDER BY key ASC
	`
	rows, err := s.db.QueryContext(ctx, query, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to query objects: %w", err)
	}
	defer rows.Close()

	var objects []Object
	for rows.Next() {
		var obj Object
		err := rows.Scan(
			&obj.ID,
			&obj.BucketName,
			&obj.Key,
			&obj.PhysicalID,
			&obj.SizeBytes,
			&obj.ContentType,
			&obj.CreatedAt,
			&obj.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan object row: %w", err)
		}
		objects = append(objects, obj)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("object rows iteration error: %w", err)
	}

	return objects, nil
}
