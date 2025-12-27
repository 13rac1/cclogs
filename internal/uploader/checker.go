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
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// ListRemoteFiles fetches all objects under a given prefix and returns a map of S3 key to file size.
// This allows efficient batch checking of multiple files with a single API call (or a few calls with pagination).
// Returns an empty map if no objects exist under the prefix.
func ListRemoteFiles(ctx context.Context, client s3ClientInterface, bucket, prefix string) (map[string]int64, error) {
	remoteFiles := make(map[string]int64)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	for {
		output, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list objects with prefix %s: %w", prefix, err)
		}

		for _, obj := range output.Contents {
			if obj.Key != nil && obj.Size != nil {
				remoteFiles[*obj.Key] = *obj.Size
			}
		}

		if !aws.ToBool(output.IsTruncated) {
			break
		}

		input.ContinuationToken = output.NextContinuationToken
	}

	return remoteFiles, nil
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
