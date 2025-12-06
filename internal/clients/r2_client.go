package clients

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	client     *s3.Client
	bucketName string
}

func NewR2Client(accountID, accessKeyID, secretKey, bucketName, endpoint string) (*R2Client, error) {
	// Если endpoint не указан, используем стандартный для R2
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: "auto",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &R2Client{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadAvatar загружает аватарку в R2
func (r *R2Client) UploadAvatar(ctx context.Context, guid string, file io.Reader, contentType string, size int64) error {
	key := fmt.Sprintf("avatars/%s", guid)

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucketName),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})

	if err != nil {
		return fmt.Errorf("failed to upload avatar to R2: %w", err)
	}

	return nil
}

// GetAvatarURL получает URL для доступа к аватарке
func (r *R2Client) GetAvatarURL(guid string) string {
	// Для R2 обычно используется публичный URL или presigned URL
	// Здесь возвращаем путь, который можно использовать для генерации presigned URL
	key := fmt.Sprintf("avatars/%s", guid)
	return key
}

// GetAvatarPresignedURL генерирует presigned URL для доступа к аватарке
func (r *R2Client) GetAvatarPresignedURL(ctx context.Context, guid string, expiresIn int64) (string, error) {
	key := fmt.Sprintf("avatars/%s", guid)

	presignClient := s3.NewPresignClient(r.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiresIn) * time.Second
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// DeleteAvatar удаляет аватарку из R2
func (r *R2Client) DeleteAvatar(ctx context.Context, guid string) error {
	key := fmt.Sprintf("avatars/%s", guid)

	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete avatar from R2: %w", err)
	}

	return nil
}
