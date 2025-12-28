package config

import (
	"context"
	"fmt"

	"github.com/13rac1/cclogs/internal/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewS3Client creates an S3 client from the provided configuration.
// Authentication priority: static credentials > AWS profile > default credential chain.
func NewS3Client(ctx context.Context, cfg *types.Config) (*s3.Client, error) {
	var opts []func(*config.LoadOptions) error

	opts = append(opts,
		config.WithRegion(cfg.S3.Region),
		config.WithRetryMaxAttempts(3),
		config.WithRetryMode(aws.RetryModeStandard),
	)

	// Use static credentials if provided (highest priority)
	if cfg.Auth.AccessKeyID != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.Auth.AccessKeyID,
				cfg.Auth.SecretAccessKey,
				cfg.Auth.SessionToken,
			),
		))
	} else if cfg.Auth.Profile != "" {
		// Use profile if no static credentials
		opts = append(opts, config.WithSharedConfigProfile(cfg.Auth.Profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	// Create S3 client with optional customizations
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3.Endpoint)
		}
		if cfg.S3.ForcePathStyle {
			o.UsePathStyle = true
		}
	})

	return client, nil
}
