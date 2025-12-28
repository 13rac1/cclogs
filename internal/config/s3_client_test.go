package config

import (
	"context"
	"testing"

	"github.com/13rac1/cclogs/internal/types"
)

func TestNewS3Client(t *testing.T) {
	// Note: These tests verify S3 client creation without making actual AWS calls.
	// Actual S3 connectivity requires valid AWS credentials and is tested in integration tests.

	tests := []struct {
		name      string
		cfg       *types.Config
		wantError bool
	}{
		{
			name: "minimal config with region",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Region: "us-west-2",
				},
			},
			wantError: false,
		},
		{
			name: "config with static credentials",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Region: "us-east-1",
				},
				Auth: types.AuthConfig{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
			wantError: false,
		},
		{
			name: "config with custom endpoint",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket:   "test-bucket",
					Region:   "us-west-2",
					Endpoint: "https://s3.us-west-002.backblazeb2.com",
				},
			},
			wantError: false,
		},
		{
			name: "config with force path style",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket:         "test-bucket",
					Region:         "us-west-2",
					ForcePathStyle: true,
				},
			},
			wantError: false,
		},
		{
			name: "config with custom endpoint and force path style",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket:         "test-bucket",
					Region:         "us-west-2",
					Endpoint:       "https://minio.example.com:9000",
					ForcePathStyle: true,
				},
			},
			wantError: false,
		},
		{
			name: "config with session token",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Region: "us-west-2",
				},
				Auth: types.AuthConfig{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					SessionToken:    "FwoGZXIvYXdzEHoaDN...",
				},
			},
			wantError: false,
		},
		{
			name: "static credentials take precedence over profile",
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Region: "us-west-2",
				},
				Auth: types.AuthConfig{
					Profile:         "nonexistent-profile",
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
			// Should succeed despite invalid profile because static credentials override
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			client, err := NewS3Client(ctx, tt.cfg)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewS3Client() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewS3Client() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("NewS3Client() returned nil client")
			}
		})
	}
}
