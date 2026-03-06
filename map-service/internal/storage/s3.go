package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewS3Storage(ctx context.Context) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to load SDK config: %v", err)
	}

	bucketName := os.Getenv("AWS_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("AWS_BUCKET_NAME must be set")
	}

	client := s3.NewFromConfig(cfg)

	return &S3Storage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucketName:    bucketName,
	}, nil
}

func (s *S3Storage) UploadToS3(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(fileHeader.Filename)
	objectKey := uuid.New().String() + ext

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to upload file: %v", err)
	}
	return objectKey, nil
}

func (s *S3Storage) GeneratePresignedURL(ctx context.Context, objectKey string) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return req.URL, nil
}

func (s *S3Storage) DeleteFromS3(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return nil
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %v", err)
	}

	return nil
}
