package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	BucketName      string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

// DefaultS3Config returns default S3 configuration from environment variables
func DefaultS3Config() *S3Config {
	return &S3Config{
		BucketName:      getEnvWithDefault("S3_ARCHIVE_BUCKET", "audit-log-archives"),
		Region:          getEnvWithDefault("AWS_REGION", "us-east-1"),
		Endpoint:        getEnvWithDefault("AWS_ENDPOINT_URL", ""),
		AccessKeyID:     getEnvWithDefault("AWS_ACCESS_KEY_ID", "dummy"),
		SecretAccessKey: getEnvWithDefault("AWS_SECRET_ACCESS_KEY", "dummy"),
	}
}

// GetClient creates and returns an S3 client
func (c *S3Config) GetClient(ctx context.Context) (*s3.Client, error) {
	var options []func(*awsconfig.LoadOptions) error
	options = append(options, awsconfig.WithRegion(c.Region))

	// Add custom endpoint resolver if endpoint is specified (for LocalStack)
	if c.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           c.Endpoint,
					SigningRegion: c.Region,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		options = append(options, awsconfig.WithEndpointResolverWithOptions(customResolver))

		// For LocalStack, use static credentials
		options = append(options, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			"",
		)))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, err
	}

	// Create S3 client with path-style addressing for LocalStack
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Force path-style addressing when using custom endpoint (LocalStack)
		if c.Endpoint != "" {
			o.UsePathStyle = true
		}
	})

	return s3Client, nil
}
