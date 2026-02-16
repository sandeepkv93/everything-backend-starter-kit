package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	maxAvatarSize    = 5 * 1024 * 1024 // 5 MB
	avatarObjectTTL  = 7 * 24 * time.Hour
	presignedURLTTL  = 15 * time.Minute
	avatarPathPrefix = "avatars"
)

var (
	ErrFileTooBig           = errors.New("file size exceeds 5MB limit")
	ErrInvalidFileType      = errors.New("invalid file type, only JPEG and PNG images are allowed")
	ErrBucketCreationFailed = errors.New("failed to create storage bucket")
	ErrUploadFailed         = errors.New("failed to upload file")
	ErrDeleteFailed         = errors.New("failed to delete file")
	ErrURLGenerationFailed  = errors.New("failed to generate presigned URL")

	allowedContentTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
	}
)

// StorageService defines the interface for object storage operations.
type StorageService interface {
	// UploadAvatar uploads a user's avatar and returns the object key.
	UploadAvatar(ctx context.Context, userID uint, file io.Reader, fileSize int64, contentType string) (string, error)

	// DeleteAvatar deletes a user's avatar by object key.
	DeleteAvatar(ctx context.Context, objectKey string) error

	// GenerateAvatarURL generates a presigned URL for avatar access.
	GenerateAvatarURL(ctx context.Context, objectKey string) (string, error)
}

// MinIOStorageService implements StorageService using MinIO/S3-compatible storage.
type MinIOStorageService struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOStorageService creates a MinIO-backed storage service.
// It ensures the bucket exists and has proper lifecycle policies.
func NewMinIOStorageService(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOStorageService, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	svc := &MinIOStorageService{
		client:     client,
		bucketName: bucketName,
	}

	if err := svc.ensureBucketExists(context.Background()); err != nil {
		return nil, err
	}

	return svc, nil
}

// ensureBucketExists creates the bucket if it doesn't exist.
func (s *MinIOStorageService) ensureBucketExists(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("%w: check bucket existence: %v", ErrBucketCreationFailed, err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("%w: create bucket: %v", ErrBucketCreationFailed, err)
		}
	}

	return nil
}

// UploadAvatar uploads a user's avatar with validation.
func (s *MinIOStorageService) UploadAvatar(ctx context.Context, userID uint, file io.Reader, fileSize int64, contentType string) (string, error) {
	// Validate file size
	if fileSize > maxAvatarSize {
		return "", ErrFileTooBig
	}

	// Validate content type
	normalizedContentType := strings.ToLower(strings.TrimSpace(contentType))
	if _, allowed := allowedContentTypes[normalizedContentType]; !allowed {
		return "", ErrInvalidFileType
	}

	// Generate unique object key with user namespace
	fileExt := contentTypeToExtension(normalizedContentType)
	objectKey := fmt.Sprintf("%s/user-%d/%s%s", avatarPathPrefix, userID, uuid.New().String(), fileExt)

	// Prepare metadata
	metadata := map[string]string{
		"Content-Type": normalizedContentType,
		"User-ID":      fmt.Sprintf("%d", userID),
		"Uploaded-At":  time.Now().UTC().Format(time.RFC3339),
	}

	// Upload file
	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, file, fileSize, minio.PutObjectOptions{
		ContentType:  normalizedContentType,
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	return objectKey, nil
}

// DeleteAvatar deletes an avatar object.
func (s *MinIOStorageService) DeleteAvatar(ctx context.Context, objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return nil // No-op for empty keys
	}

	err := s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	return nil
}

// GenerateAvatarURL generates a presigned GET URL for avatar access.
func (s *MinIOStorageService) GenerateAvatarURL(ctx context.Context, objectKey string) (string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return "", fmt.Errorf("%w: empty object key", ErrURLGenerationFailed)
	}

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectKey, presignedURLTTL, url.Values{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrURLGenerationFailed, err)
	}

	return presignedURL.String(), nil
}

// contentTypeToExtension maps content type to file extension.
func contentTypeToExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}
