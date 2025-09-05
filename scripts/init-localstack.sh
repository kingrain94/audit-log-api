#!/bin/bash

# Wait for LocalStack to be ready
# echo "Waiting for LocalStack to be ready..."
# while ! curl -s http://localhost:4566/_localstack/health | grep -q '"sqs": "running"'; do
#     sleep 1
# done

echo "Initializing LocalStack services..."

# Create SQS queues
echo "Creating SQS queues..."

# Create index queue (for log indexing operations)
echo "Creating audit-log-index-queue..."
aws --endpoint-url=http://localhost:4566 sqs create-queue \
    --queue-name audit-log-index-queue \
    --attributes '{
        "VisibilityTimeout": "30",
        "MessageRetentionPeriod": "86400",
        "DelaySeconds": "0",
        "ReceiveMessageWaitTimeSeconds": "20"
    }'

# Create archive queue (for log archival operations)
echo "Creating audit-log-archive-queue..."
aws --endpoint-url=http://localhost:4566 sqs create-queue \
    --queue-name audit-log-archive-queue \
    --attributes '{
        "VisibilityTimeout": "60",
        "MessageRetentionPeriod": "86400",
        "DelaySeconds": "0",
        "ReceiveMessageWaitTimeSeconds": "20"
    }'

# Create cleanup queue (for log cleanup operations)
echo "Creating audit-log-cleanup-queue..."
aws --endpoint-url=http://localhost:4566 sqs create-queue \
    --queue-name audit-log-cleanup-queue \
    --attributes '{
        "VisibilityTimeout": "60",
        "MessageRetentionPeriod": "86400",
        "DelaySeconds": "0",
        "ReceiveMessageWaitTimeSeconds": "20"
    }'

# Create S3 buckets
echo "Creating S3 buckets..."

# Create main archive bucket
echo "Creating audit-log-archives bucket..."
aws --endpoint-url=http://localhost:4566 s3 mb s3://audit-log-archives

