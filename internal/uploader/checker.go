package uploader

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// s3ClientInterface defines the minimal S3 client interface needed for checking file existence.
type s3ClientInterface interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// ShouldUpload checks if a file should be uploaded by comparing with remote.
// Returns true if file should be uploaded (missing or different).
// Returns false if file should be skipped (exists and identical).
func ShouldUpload(ctx context.Context, client s3ClientInterface, bucket, key string, localSize int64) (bool, error) {
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return true, nil
		}
		return false, fmt.Errorf("head object %s: %w", key, err)
	}

	if head.ContentLength == nil {
		return true, nil
	}

	if *head.ContentLength != localSize {
		return true, nil
	}

	return false, nil
}
