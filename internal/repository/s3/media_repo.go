package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

var _ repository.MediaRepository = (*MediaRepo)(nil)

const awsErrNotFound = "NotFound"

type MediaRepo struct {
	client     *s3.S3
	bucketName string
}

func NewMediaRepo(client *s3.S3, bucketName string) *MediaRepo {
	return &MediaRepo{
		client:     client,
		bucketName: bucketName,
	}
}

func (r *MediaRepo) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := r.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("uploading media: %w", err)
	}
	return nil
}

func (r *MediaRepo) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchKey || awsErr.Code() == awsErrNotFound {
				return nil, repository.ErrNotFound
			}
		}
		return nil, fmt.Errorf("downloading media: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("reading media: %w", err)
	}
	return data, nil
}

func (r *MediaRepo) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting media: %w", err)
	}
	return nil
}

func (r *MediaRepo) Exists(ctx context.Context, key string) (bool, error) {
	_, err := r.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchKey || awsErr.Code() == awsErrNotFound {
				return false, repository.ErrNotFound
			}
		}
		return false, fmt.Errorf("checking media existence: %w", err)
	}
	return true, nil
}

func (r *MediaRepo) GenerateUploadURL(_ context.Context, key string, expires time.Duration) (string, error) {
	req, _ := r.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("generating upload url: %w", err)
	}
	return url, nil
}

func (r *MediaRepo) GenerateViewURL(_ context.Context, key string, expires time.Duration) (string, error) {
	req, _ := r.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("generating view url: %w", err)
	}
	return url, nil
}
