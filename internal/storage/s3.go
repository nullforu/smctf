package storage

import (
	"context"
	"errors"
	"maps"
	"time"

	"smctf/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const defaultPresignTTL = 15 * time.Minute

type S3ChallengeFileStore struct {
	bucket     string
	presignTTL time.Duration
	client     *s3.Client
	presigner  *s3.PresignClient
}

func NewS3ChallengeFileStore(ctx context.Context, cfg config.S3Config) (*S3ChallengeFileStore, error) {
	if !cfg.Enabled {
		return nil, ErrNotConfigured
	}

	if cfg.Bucket == "" {
		return nil, errors.New("S3_BUCKET must not be empty")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return nil, err
	}

	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
			return nil, errors.New("S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY must both be set")
		}

		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}

		o.UsePathStyle = cfg.ForcePathStyle
	})

	presignTTL := cfg.PresignTTL
	if presignTTL <= 0 {
		presignTTL = defaultPresignTTL
	}

	return &S3ChallengeFileStore{
		bucket:     cfg.Bucket,
		presignTTL: presignTTL,
		client:     client,
		presigner:  s3.NewPresignClient(client),
	}, nil
}

func (s *S3ChallengeFileStore) PresignUpload(ctx context.Context, key, contentType string) (PresignedPost, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	opts := func(o *s3.PresignPostOptions) {
		o.Expires = s.presignTTL
		o.Conditions = []any{
			map[string]string{"Content-Type": contentType},
		}
	}

	resp, err := s.presigner.PresignPostObject(ctx, input, opts)
	if err != nil {
		return PresignedPost{}, err
	}

	fields := make(map[string]string, len(resp.Values)+1)
	maps.Copy(fields, resp.Values)
	fields["Content-Type"] = contentType

	return PresignedPost{
		URL:       resp.URL,
		Fields:    fields,
		ExpiresAt: time.Now().UTC().Add(s.presignTTL),
	}, nil
}

func (s *S3ChallengeFileStore) PresignDownload(ctx context.Context, key, filename string) (PresignedURL, error) {
	disposition := buildContentDisposition(filename)
	resp, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disposition),
	}, func(o *s3.PresignOptions) {
		o.Expires = s.presignTTL
	})
	if err != nil {
		return PresignedURL{}, err
	}

	return PresignedURL{
		URL:       resp.URL,
		ExpiresAt: time.Now().UTC().Add(s.presignTTL),
	}, nil
}

func buildContentDisposition(filename string) string {
	if filename == "" {
		return "attachment"
	}
	escaped := escapeContentDispositionFilename(filename)
	return "attachment; filename=\"" + escaped + "\""
}

func escapeContentDispositionFilename(filename string) string {
	runes := []rune(filename)
	escaped := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r == '\\' || r == '"' {
			escaped = append(escaped, '\\', r)
			continue
		}
		escaped = append(escaped, r)
	}
	return string(escaped)
}

func (s *S3ChallengeFileStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
