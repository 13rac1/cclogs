package uploader

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3Client implements the minimal S3 client interface needed for testing.
type mockS3Client struct {
	headObjectResp *s3.HeadObjectOutput
	headObjectErr  error
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return m.headObjectResp, m.headObjectErr
}

// int64Ptr returns a pointer to an int64 value.
func int64Ptr(v int64) *int64 {
	return &v
}

func TestShouldUpload(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockS3Client)
		localSize int64
		want      bool
		wantErr   bool
	}{
		{
			name: "file doesn't exist (NoSuchKey) - should upload",
			setupMock: func(m *mockS3Client) {
				m.headObjectErr = &types.NoSuchKey{}
			},
			localSize: 1024,
			want:      true,
			wantErr:   false,
		},
		{
			name: "file doesn't exist (NotFound) - should upload",
			setupMock: func(m *mockS3Client) {
				m.headObjectErr = &types.NotFound{}
			},
			localSize: 1024,
			want:      true,
			wantErr:   false,
		},
		{
			name: "file exists with same size - should skip",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: int64Ptr(1024),
				}
			},
			localSize: 1024,
			want:      false,
			wantErr:   false,
		},
		{
			name: "file exists with different size - should upload",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: int64Ptr(2048),
				}
			},
			localSize: 1024,
			want:      true,
			wantErr:   false,
		},
		{
			name: "file exists with larger size - should upload",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: int64Ptr(512),
				}
			},
			localSize: 1024,
			want:      true,
			wantErr:   false,
		},
		{
			name: "file exists but no ContentLength - should upload to be safe",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: nil,
				}
			},
			localSize: 1024,
			want:      true,
			wantErr:   false,
		},
		{
			name: "permission error - should return error",
			setupMock: func(m *mockS3Client) {
				m.headObjectErr = errors.New("access denied")
			},
			localSize: 1024,
			want:      false,
			wantErr:   true,
		},
		{
			name: "network error - should return error",
			setupMock: func(m *mockS3Client) {
				m.headObjectErr = errors.New("connection timeout")
			},
			localSize: 1024,
			want:      false,
			wantErr:   true,
		},
		{
			name: "zero size file exists and matches - should skip",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: int64Ptr(0),
				}
			},
			localSize: 0,
			want:      false,
			wantErr:   false,
		},
		{
			name: "large file exists and matches - should skip",
			setupMock: func(m *mockS3Client) {
				m.headObjectResp = &s3.HeadObjectOutput{
					ContentLength: int64Ptr(1073741824), // 1 GB
				}
			},
			localSize: 1073741824,
			want:      false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockS3Client{}
			tt.setupMock(mock)

			got, err := ShouldUpload(context.Background(), mock, "test-bucket", "test-key", tt.localSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("ShouldUpload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ShouldUpload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUploadContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mock := &mockS3Client{
		headObjectErr: context.Canceled,
	}

	_, err := ShouldUpload(ctx, mock, "test-bucket", "test-key", 1024)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}
