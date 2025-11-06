package s3

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/paragkamble/s3bench/internal/metrics"
	"go.uber.org/zap"
)

// Client wraps the AWS S3 client with additional functionality
type Client struct {
	s3Client *s3.Client
	bucket   string
	logger   *zap.Logger
	metrics  *metrics.Metrics
}

// ClientConfig holds configuration for creating an S3 client
type ClientConfig struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PathStyle     bool
	SkipTLSVerify bool
	Logger        *zap.Logger
	Metrics       *metrics.Metrics
}

// NewClient creates a new S3 client
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	var opts []func(*config.LoadOptions) error

	// Region
	opts = append(opts, config.WithRegion(cfg.Region))

	// Credentials
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Custom HTTP client for TLS verification
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if cfg.SkipTLSVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Create S3 client options
	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.HTTPClient = httpClient
		},
	}

	// Custom endpoint
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	// Path-style addressing
	if cfg.PathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	s3Client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &Client{
		s3Client: s3Client,
		bucket:   cfg.Bucket,
		logger:   cfg.Logger,
		metrics:  cfg.Metrics,
	}, nil
}

// Check performs a health check by doing a HEAD bucket operation
func (c *Client) Check(ctx context.Context) error {
	_, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("bucket check failed: %w", err)
	}
	return nil
}

// CreateBucket creates the bucket if it doesn't exist
func (c *Client) CreateBucket(ctx context.Context) error {
	_, err := c.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})

	if err != nil {
		// Ignore if bucket already exists
		var bae *types.BucketAlreadyExists
		var baoyoe *types.BucketAlreadyOwnedByYou
		if !(errors.As(err, &bae) || errors.As(err, &baoyoe)) {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	c.logger.Info("bucket ready", zap.String("bucket", c.bucket))
	return nil
}

// SetVersioning enables or disables versioning on the bucket
func (c *Client) SetVersioning(ctx context.Context, enabled bool) error {
	var status types.BucketVersioningStatus
	if enabled {
		status = types.BucketVersioningStatusEnabled
	} else {
		status = types.BucketVersioningStatusSuspended
	}

	_, err := c.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(c.bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: status,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to set versioning: %w", err)
	}

	return nil
}

// PutObject uploads an object to S3
func (c *Client) PutObject(ctx context.Context, key string, body io.Reader, size int64, metadata map[string]string) error {
	start := time.Now()

	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		Metadata:      metadata,
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpPut), string(metrics.StatusError), duration)
		return fmt.Errorf("put failed: %w", err)
	}

	c.metrics.RecordOp(string(metrics.OpPut), string(metrics.StatusSuccess), duration)
	c.metrics.RecordBytesWritten(size)

	c.logger.Debug("put object",
		zap.String("key", key),
		zap.Int64("size", size),
		zap.Duration("latency", duration),
	)

	return nil
}

// GetObject downloads an object from S3
func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, map[string]string, int64, error) {
	start := time.Now()

	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpGet), string(metrics.StatusError), duration)
		return nil, nil, 0, fmt.Errorf("get failed: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	c.metrics.RecordOp(string(metrics.OpGet), string(metrics.StatusSuccess), duration)
	c.metrics.RecordBytesRead(size)

	c.logger.Debug("get object",
		zap.String("key", key),
		zap.Int64("size", size),
		zap.Duration("latency", duration),
	)

	return result.Body, result.Metadata, size, nil
}

// DeleteObject deletes an object from S3
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	start := time.Now()

	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpDelete), string(metrics.StatusError), duration)
		return fmt.Errorf("delete failed: %w", err)
	}

	c.metrics.RecordOp(string(metrics.OpDelete), string(metrics.StatusSuccess), duration)

	c.logger.Debug("delete object",
		zap.String("key", key),
		zap.Duration("latency", duration),
	)

	return nil
}

// CopyObject copies an object within or across buckets
func (c *Client) CopyObject(ctx context.Context, srcKey, dstKey, dstBucket string) error {
	start := time.Now()

	if dstBucket == "" {
		dstBucket = c.bucket
	}

	copySource := fmt.Sprintf("%s/%s", c.bucket, srcKey)

	_, err := c.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpCopy), string(metrics.StatusError), duration)
		return fmt.Errorf("copy failed: %w", err)
	}

	c.metrics.RecordOp(string(metrics.OpCopy), string(metrics.StatusSuccess), duration)

	c.logger.Debug("copy object",
		zap.String("src_key", srcKey),
		zap.String("dst_key", dstKey),
		zap.String("dst_bucket", dstBucket),
		zap.Duration("latency", duration),
	)

	return nil
}

// HeadObject retrieves object metadata without downloading
func (c *Client) HeadObject(ctx context.Context, key string) (map[string]string, int64, error) {
	start := time.Now()

	result, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpHead), string(metrics.StatusError), duration)
		return nil, 0, fmt.Errorf("head failed: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	c.metrics.RecordOp(string(metrics.OpHead), string(metrics.StatusSuccess), duration)

	c.logger.Debug("head object",
		zap.String("key", key),
		zap.Int64("size", size),
		zap.Duration("latency", duration),
	)

	return result.Metadata, size, nil
}

// ListObjects lists objects with a given prefix
func (c *Client) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	start := time.Now()

	result, err := c.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	})

	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordOp(string(metrics.OpList), string(metrics.StatusError), duration)
		return nil, fmt.Errorf("list failed: %w", err)
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	c.metrics.RecordOp(string(metrics.OpList), string(metrics.StatusSuccess), duration)

	c.logger.Debug("list objects",
		zap.String("prefix", prefix),
		zap.Int("count", len(keys)),
		zap.Duration("latency", duration),
	)

	return keys, nil
}

// DeleteObjectsByMetadata deletes objects matching specific metadata
func (c *Client) DeleteObjectsByMetadata(ctx context.Context, prefix string, metadataKey string, metadataValue string) (int, error) {
	var deleted int
	var continuationToken *string

	for {
		result, err := c.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})

		if err != nil {
			return deleted, fmt.Errorf("list failed: %w", err)
		}

		for _, obj := range result.Contents {
			if obj.Key == nil {
				continue
			}

			// Check metadata
			head, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(c.bucket),
				Key:    obj.Key,
			})

			if err != nil {
				c.logger.Warn("failed to head object during cleanup",
					zap.String("key", *obj.Key),
					zap.Error(err),
				)
				continue
			}

			// Check if metadata matches
			if val, ok := head.Metadata[metadataKey]; ok && val == metadataValue {
				if err := c.DeleteObject(ctx, *obj.Key); err != nil {
					c.logger.Warn("failed to delete object during cleanup",
						zap.String("key", *obj.Key),
						zap.Error(err),
					)
					continue
				}
				deleted++
			}
		}

		if !aws.ToBool(result.IsTruncated) {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return deleted, nil
}
