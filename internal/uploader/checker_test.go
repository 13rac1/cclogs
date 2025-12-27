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
	headObjectResp    *s3.HeadObjectOutput
	headObjectErr     error
	listObjectsV2Resp *s3.ListObjectsV2Output
	listObjectsV2Err  error
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return m.headObjectResp, m.headObjectErr
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.listObjectsV2Resp, m.listObjectsV2Err
}

// int64Ptr returns a pointer to an int64 value.
func int64Ptr(v int64) *int64 {
	return &v
}

// stringPtr returns a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
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

func TestListRemoteFiles(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mockS3Client)
		bucket    string
		prefix    string
		want      map[string]int64
		wantErr   bool
	}{
		{
			name: "empty bucket - returns empty map",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Resp = &s3.ListObjectsV2Output{
					IsTruncated: boolPtr(false),
					Contents:    []types.Object{},
				}
			},
			bucket: "test-bucket",
			prefix: "project-a/",
			want:   map[string]int64{},
		},
		{
			name: "single file",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Resp = &s3.ListObjectsV2Output{
					IsTruncated: boolPtr(false),
					Contents: []types.Object{
						{
							Key:  stringPtr("project-a/session.jsonl"),
							Size: int64Ptr(1024),
						},
					},
				}
			},
			bucket: "test-bucket",
			prefix: "project-a/",
			want: map[string]int64{
				"project-a/session.jsonl": 1024,
			},
		},
		{
			name: "multiple files",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Resp = &s3.ListObjectsV2Output{
					IsTruncated: boolPtr(false),
					Contents: []types.Object{
						{
							Key:  stringPtr("project-a/session1.jsonl"),
							Size: int64Ptr(1024),
						},
						{
							Key:  stringPtr("project-a/session2.jsonl"),
							Size: int64Ptr(2048),
						},
						{
							Key:  stringPtr("project-a/nested/session3.jsonl"),
							Size: int64Ptr(512),
						},
					},
				}
			},
			bucket: "test-bucket",
			prefix: "project-a/",
			want: map[string]int64{
				"project-a/session1.jsonl":        1024,
				"project-a/session2.jsonl":        2048,
				"project-a/nested/session3.jsonl": 512,
			},
		},
		{
			name: "object with nil key - skipped",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Resp = &s3.ListObjectsV2Output{
					IsTruncated: boolPtr(false),
					Contents: []types.Object{
						{
							Key:  nil,
							Size: int64Ptr(1024),
						},
						{
							Key:  stringPtr("project-a/session.jsonl"),
							Size: int64Ptr(2048),
						},
					},
				}
			},
			bucket: "test-bucket",
			prefix: "project-a/",
			want: map[string]int64{
				"project-a/session.jsonl": 2048,
			},
		},
		{
			name: "object with nil size - skipped",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Resp = &s3.ListObjectsV2Output{
					IsTruncated: boolPtr(false),
					Contents: []types.Object{
						{
							Key:  stringPtr("project-a/session1.jsonl"),
							Size: nil,
						},
						{
							Key:  stringPtr("project-a/session2.jsonl"),
							Size: int64Ptr(2048),
						},
					},
				}
			},
			bucket: "test-bucket",
			prefix: "project-a/",
			want: map[string]int64{
				"project-a/session2.jsonl": 2048,
			},
		},
		{
			name: "api error",
			setupMock: func(m *mockS3Client) {
				m.listObjectsV2Err = errors.New("access denied")
			},
			bucket:  "test-bucket",
			prefix:  "project-a/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockS3Client{}
			tt.setupMock(mock)

			got, err := ListRemoteFiles(context.Background(), mock, tt.bucket, tt.prefix)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListRemoteFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ListRemoteFiles() returned %d files, want %d", len(got), len(tt.want))
			}

			for key, size := range tt.want {
				gotSize, exists := got[key]
				if !exists {
					t.Errorf("ListRemoteFiles() missing key %q", key)
					continue
				}
				if gotSize != size {
					t.Errorf("ListRemoteFiles() key %q size = %d, want %d", key, gotSize, size)
				}
			}
		})
	}
}

func TestListRemoteFilesPagination(t *testing.T) {
	// Test pagination by creating a mock that tracks calls
	callCount := 0
	mock := &paginatingMockS3Client{
		callCount: &callCount,
	}

	got, err := ListRemoteFiles(context.Background(), mock, "test-bucket", "project-a/")
	if err != nil {
		t.Fatalf("ListRemoteFiles() failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls for pagination, got %d", callCount)
	}

	want := map[string]int64{
		"project-a/session1.jsonl": 1024,
		"project-a/session2.jsonl": 2048,
	}

	if len(got) != len(want) {
		t.Errorf("ListRemoteFiles() returned %d files, want %d", len(got), len(want))
	}

	for key, size := range want {
		gotSize, exists := got[key]
		if !exists {
			t.Errorf("ListRemoteFiles() missing key %q", key)
			continue
		}
		if gotSize != size {
			t.Errorf("ListRemoteFiles() key %q size = %d, want %d", key, gotSize, size)
		}
	}
}

// paginatingMockS3Client simulates pagination for testing
type paginatingMockS3Client struct {
	callCount *int
}

func (m *paginatingMockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("not implemented")
}

func (m *paginatingMockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	*m.callCount++
	if *m.callCount == 1 {
		// First page
		return &s3.ListObjectsV2Output{
			IsTruncated: boolPtr(true),
			Contents: []types.Object{
				{
					Key:  stringPtr("project-a/session1.jsonl"),
					Size: int64Ptr(1024),
				},
			},
			NextContinuationToken: stringPtr("token1"),
		}, nil
	}
	// Second page
	return &s3.ListObjectsV2Output{
		IsTruncated: boolPtr(false),
		Contents: []types.Object{
			{
				Key:  stringPtr("project-a/session2.jsonl"),
				Size: int64Ptr(2048),
			},
		},
	}, nil
}
