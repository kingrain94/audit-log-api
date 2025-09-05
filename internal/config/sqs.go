package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSConfig struct {
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	IndexQueueURL   string `mapstructure:"index_queue_url"`
	ArchiveQueueURL string `mapstructure:"archive_queue_url"`
	CleanupQueueURL string `mapstructure:"cleanup_queue_url"`
}

func DefaultSQSConfig() *SQSConfig {
	return &SQSConfig{
		Region:          getEnvOrDefault("AWS_REGION", "us-east-1"),
		Endpoint:        getEnvOrDefault("AWS_SQS_ENDPOINT", "http://localhost:4566"),
		AccessKeyID:     getEnvOrDefault("AWS_ACCESS_KEY_ID", "dummy"),
		SecretAccessKey: getEnvOrDefault("AWS_SECRET_ACCESS_KEY", "dummy"),
		IndexQueueURL:   getEnvOrDefault("AWS_SQS_INDEX_QUEUE_URL", "http://localhost:4566/000000000000/audit-log-index-queue"),
		ArchiveQueueURL: getEnvOrDefault("AWS_SQS_ARCHIVE_QUEUE_URL", "http://localhost:4566/000000000000/audit-log-archive-queue"),
		CleanupQueueURL: getEnvOrDefault("AWS_SQS_CLEANUP_QUEUE_URL", "http://localhost:4566/000000000000/audit-log-cleanup-queue"),
	}
}

func (c *SQSConfig) GetClient() (*sqs.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sqs.ServiceID {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           c.Endpoint,
				SigningRegion: c.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(c.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return sqs.NewFromConfig(cfg), nil
}
