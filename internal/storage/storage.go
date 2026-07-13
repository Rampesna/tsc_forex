// Package storage provides an S3/MinIO-backed object store for user avatars and
// other shared assets. Running it as a shared service means an image uploaded
// through one pod is immediately readable from every other pod.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// Store abstracts object storage so handlers don't depend on MinIO directly.
type Store interface {
	// Put uploads an object and returns its public URL.
	Put(ctx context.Context, key, contentType string, data []byte) (string, error)
}

// MinioStore is the production S3/MinIO implementation.
type MinioStore struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// New builds a MinIO store and ensures the bucket exists.
func New(cfg config.StorageConfig) (*MinioStore, error) {
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage: endpoint/access_key/secret_key required")
	}
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: bucket check: %w", err)
	}
	if !exists {
		if err := cli.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("storage: make bucket: %w", err)
		}
	}

	// Avatars are served publicly via the CDN URL, so grant anonymous read on
	// the bucket's objects (idempotent). Non-fatal if the backend lacks perms.
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, cfg.Bucket)
	if err := cli.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		// Log-free here (no logger dependency); handler-level upload still works
		// and ops can set the policy manually if this is denied.
		_ = err
	}

	base := cfg.PublicBaseURL
	if base == "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		base = fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket)
	}

	return &MinioStore{client: cli, bucket: cfg.Bucket, publicBaseURL: base}, nil
}

// Put uploads data under key and returns its public URL.
func (s *MinioStore) Put(ctx context.Context, key, contentType string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType, CacheControl: "public, max-age=86400"})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", s.publicBaseURL, key), nil
}

// Compile-time interface check.
var _ Store = (*MinioStore)(nil)
var _ io.Reader = (*bytes.Reader)(nil)
