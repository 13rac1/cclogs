package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Client defines the minimal S3 client interface needed for manifest operations.
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Load downloads and parses the manifest from S3.
// Returns an empty manifest if the file doesn't exist (first run).
// Returns an error for other failures (network, permissions, corrupt JSON).
func Load(ctx context.Context, client S3Client, bucket, key string) (*Manifest, error) {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return New(), nil
		}
		return nil, fmt.Errorf("downloading manifest: %w", err)
	}
	defer func() { _ = output.Body.Close() }()

	var m Manifest
	if err := json.NewDecoder(output.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}

	if m.Version != 1 {
		return nil, fmt.Errorf("unsupported manifest version: %d", m.Version)
	}

	if m.Files == nil {
		m.Files = make(map[string]FileEntry)
	}

	return &m, nil
}

// Save uploads the manifest to S3 as JSON.
func Save(ctx context.Context, client S3Client, bucket, key string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return fmt.Errorf("uploading manifest: %w", err)
	}

	return nil
}
