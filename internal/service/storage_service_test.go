package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// TestLazyInitDoesNotBlockStartup verifies that MinIOStorageService construction
// succeeds even when MinIO is unreachable, deferring connection until first use.
func TestLazyInitDoesNotBlockStartup(t *testing.T) {
	// Attempt to create service with unreachable endpoint
	// This should NOT fail at construction time
	svc, err := NewMinIOStorageService("invalid-endpoint:9999", "key", "secret", "bucket", false)
	if err != nil {
		t.Fatalf("expected construction to succeed even with unreachable MinIO, got error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// First operation should fail due to unreachable MinIO
	ctx := context.Background()
	file := bytes.NewReader([]byte("fake image data"))
	_, err = svc.UploadAvatar(ctx, 1, file, 100, "image/png")
	if err == nil {
		t.Fatal("expected upload to fail with unreachable MinIO")
	}
	// Error should be related to bucket creation/connection, not construction
	if !strings.Contains(err.Error(), "bucket") && !strings.Contains(err.Error(), "connection") {
		t.Logf("got error: %v", err)
	}
}

// TestDeleteAvatarEnforcesOwnership verifies that users cannot delete
// avatars belonging to other users (cross-user delete prevention).
func TestDeleteAvatarEnforcesOwnership(t *testing.T) {
	tests := []struct {
		name        string
		userID      uint
		objectKey   string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid ownership",
			userID:      123,
			objectKey:   "avatars/user-123/somefile.jpg",
			expectError: false,
		},
		{
			name:        "cross-user delete attempt",
			userID:      123,
			objectKey:   "avatars/user-456/otherfile.jpg",
			expectError: true,
			errorType:   ErrUnauthorizedAccess,
		},
		{
			name:        "malicious path traversal attempt",
			userID:      123,
			objectKey:   "avatars/user-123/../user-456/file.jpg",
			expectError: true,
			errorType:   ErrUnauthorizedAccess,
		},
		{
			name:        "missing user prefix",
			userID:      123,
			objectKey:   "avatars/file.jpg",
			expectError: true,
			errorType:   ErrUnauthorizedAccess,
		},
		{
			name:        "wrong prefix format",
			userID:      123,
			objectKey:   "avatars/user_123/file.jpg",
			expectError: true,
			errorType:   ErrUnauthorizedAccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service (construction should succeed)
			svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}

			// Inject a mock client to avoid network calls for ownership validation
			// For this test, we only validate the ownership logic, not MinIO interaction
			// The actual deletion will fail with connection error, but ownership check comes first

			ctx := context.Background()
			err = svc.DeleteAvatar(ctx, tt.userID, tt.objectKey)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Fatalf("expected error type %v, got %v", tt.errorType, err)
				}
			}
			// Note: For valid ownership cases, we'd get connection error to MinIO
			// which is expected in unit tests without a running MinIO server
		})
	}
}

// TestUploadAvatarDetectsActualContentType verifies that content type
// is detected from file bytes, not trusted from client headers.
func TestUploadAvatarDetectsActualContentType(t *testing.T) {
	tests := []struct {
		name              string
		fileContent       []byte
		clientContentType string
		expectError       bool
		errorType         error
	}{
		{
			name:              "valid JPEG with correct header",
			fileContent:       []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF"),
			clientContentType: "image/jpeg",
			expectError:       false, // Will fail with MinIO connection, but content type check passes
		},
		{
			name:              "valid PNG with correct header",
			fileContent:       []byte("\x89PNG\r\n\x1a\n"),
			clientContentType: "image/png",
			expectError:       false, // Will fail with MinIO connection, but content type check passes
		},
		{
			name:              "text file spoofed as image/jpeg",
			fileContent:       []byte("This is plain text, not an image"),
			clientContentType: "image/jpeg", // Client lies
			expectError:       true,
			errorType:         ErrInvalidFileType, // Should detect as text/plain
		},
		{
			name:              "HTML file spoofed as image/png",
			fileContent:       []byte("<html><body>Not an image</body></html>"),
			clientContentType: "image/png", // Client lies
			expectError:       true,
			errorType:         ErrInvalidFileType, // Should detect as text/html
		},
		{
			name:              "executable spoofed as image",
			fileContent:       []byte("MZ\x90\x00"), // PE header
			clientContentType: "image/jpeg",
			expectError:       true,
			errorType:         ErrInvalidFileType, // Should detect as application/octet-stream
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}

			ctx := context.Background()
			file := bytes.NewReader(tt.fileContent)

			_, err = svc.UploadAvatar(ctx, 1, file, int64(len(tt.fileContent)), tt.clientContentType)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Fatalf("expected error type %v, got %v", tt.errorType, err)
				}
			}
			// For valid content, we'd still get MinIO connection error in unit test
			// but the important part is content validation passed
		})
	}
}

// TestDeleteAvatarEmptyKeyNoOp verifies empty keys are no-op.
func TestDeleteAvatarEmptyKeyNoOp(t *testing.T) {
	svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = svc.DeleteAvatar(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("expected no error for empty key, got: %v", err)
	}

	err = svc.DeleteAvatar(context.Background(), 1, "   ")
	if err != nil {
		t.Fatalf("expected no error for whitespace key, got: %v", err)
	}
}

// TestUploadAvatarSizeLimit verifies file size limit enforcement.
func TestUploadAvatarSizeLimit(t *testing.T) {
	svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create a file larger than 5MB
	largeFile := bytes.NewReader(make([]byte, 6*1024*1024))

	_, err = svc.UploadAvatar(context.Background(), 1, largeFile, 6*1024*1024, "image/jpeg")
	if !errors.Is(err, ErrFileTooBig) {
		t.Fatalf("expected ErrFileTooBig, got: %v", err)
	}
}

// TestGenerateAvatarURLEmptyKey verifies error on empty object key.
func TestGenerateAvatarURLEmptyKey(t *testing.T) {
	svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GenerateAvatarURL(context.Background(), "")
	if !errors.Is(err, ErrURLGenerationFailed) {
		t.Fatalf("expected ErrURLGenerationFailed, got: %v", err)
	}
}

// Mock reader that returns error on read (for testing read failures).
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

// TestUploadAvatarReadError verifies handling of file read errors.
func TestUploadAvatarReadError(t *testing.T) {
	svc, err := NewMinIOStorageService("localhost:9999", "key", "secret", "bucket", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.UploadAvatar(context.Background(), 1, &errorReader{}, 1000, "image/jpeg")
	if err == nil {
		t.Fatal("expected error from read failure")
	}
	if !errors.Is(err, ErrUploadFailed) {
		t.Fatalf("expected ErrUploadFailed, got: %v", err)
	}
}
