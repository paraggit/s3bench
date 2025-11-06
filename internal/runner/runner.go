package runner

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paragkamble/s3bench/internal/config"
	"github.com/paragkamble/s3bench/internal/data"
	"github.com/paragkamble/s3bench/internal/metrics"
	"github.com/paragkamble/s3bench/internal/s3"
	"github.com/paragkamble/s3bench/internal/workload"
	"go.uber.org/zap"
)

// Runner orchestrates the workload execution
type Runner struct {
	cfg         *config.Config
	s3Client    *s3.Client
	generator   *data.Generator
	verifier    *data.Verifier
	scheduler   *workload.Scheduler
	keygen      *workload.KeyGenerator
	sizeDist    data.SizeDistribution
	rateLimiter workload.RateLimiter
	metrics     *metrics.Metrics
	logger      *zap.Logger

	opsCounter int64
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// New creates a new workload runner
func New(cfg *config.Config, logger *zap.Logger, m *metrics.Metrics) (*Runner, error) {
	// Create S3 client
	s3Client, err := s3.NewClient(context.Background(), s3.ClientConfig{
		Endpoint:      cfg.Endpoint,
		Region:        cfg.Region,
		Bucket:        cfg.Bucket,
		AccessKey:     cfg.AccessKey,
		SecretKey:     cfg.SecretKey,
		PathStyle:     cfg.PathStyle,
		SkipTLSVerify: cfg.SkipTLSVerify,
		Logger:        logger,
		Metrics:       m,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create data generator
	generator, err := data.NewGenerator(cfg.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create data generator: %w", err)
	}

	verifier := data.NewVerifier(generator)

	// Create operation scheduler
	scheduler, err := workload.NewScheduler(cfg.Mix, cfg.Keys, time.Now().UnixNano())
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	// Create key generator
	keygen := workload.NewKeyGenerator(cfg.Prefix, cfg.KeyTemplate, cfg.Keys)

	// Create size distribution
	sizeDist, err := data.ParseSizeDistribution(cfg.Size, time.Now().UnixNano())
	if err != nil {
		return nil, fmt.Errorf("failed to parse size distribution: %w", err)
	}

	// Create rate limiter
	rateLimiter := workload.NewRateLimiter(cfg.RateType, cfg.RateLimit, time.Now().UnixNano())

	return &Runner{
		cfg:         cfg,
		s3Client:    s3Client,
		generator:   generator,
		verifier:    verifier,
		scheduler:   scheduler,
		keygen:      keygen,
		sizeDist:    sizeDist,
		rateLimiter: rateLimiter,
		metrics:     m,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}, nil
}

// Run starts the workload
func (r *Runner) Run(ctx context.Context) error {
	// Setup bucket if needed
	if r.cfg.CreateBucket {
		if err := r.s3Client.CreateBucket(ctx); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Set versioning if requested
	if r.cfg.Versioning == "on" {
		if err := r.s3Client.SetVersioning(ctx, true); err != nil {
			r.logger.Warn("failed to enable versioning", zap.Error(err))
		}
	} else if r.cfg.Versioning == "off" {
		if err := r.s3Client.SetVersioning(ctx, false); err != nil {
			r.logger.Warn("failed to disable versioning", zap.Error(err))
		}
	}

	// Handle cleanup mode
	if r.cfg.Cleanup {
		return r.runCleanup(ctx)
	}

	// Create worker context
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create duration timer if specified
	var durationChan <-chan time.Time
	if r.cfg.Duration > 0 {
		durationChan = time.After(r.cfg.Duration)
	} else {
		// Create a channel that never fires
		durationChan = make(<-chan time.Time)
	}

	// Start workers
	r.logger.Info("starting workload",
		zap.Int("concurrency", r.cfg.Concurrency),
		zap.Duration("duration", r.cfg.Duration),
		zap.Int64("operations", r.cfg.Operations),
	)

	for i := 0; i < r.cfg.Concurrency; i++ {
		r.wg.Add(1)
		go r.worker(workerCtx, i)
	}

	// Wait for completion
	select {
	case <-workerCtx.Done():
		r.logger.Info("workload cancelled")
	case <-r.stopChan:
		r.logger.Info("workload stopped")
	case <-durationChan:
		r.logger.Info("workload duration elapsed")
		cancel()
	}

	// Wait for all workers to finish
	r.wg.Wait()

	r.logger.Info("workload completed",
		zap.Int64("total_operations", atomic.LoadInt64(&r.opsCounter)),
	)

	return nil
}

// Stop gracefully stops the workload
func (r *Runner) Stop() {
	close(r.stopChan)
}

// worker executes operations in a loop
func (r *Runner) worker(ctx context.Context, workerID int) {
	defer r.wg.Done()

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

	r.metrics.SetActiveWorkers(r.cfg.Concurrency)
	defer func() {
		r.metrics.SetActiveWorkers(0)
	}()

	for {
		// Check if we should stop
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		default:
		}

		// Check operation limit
		if r.cfg.Operations > 0 {
			current := atomic.LoadInt64(&r.opsCounter)
			if current >= r.cfg.Operations {
				return
			}
		}

		// Rate limiting
		if err := r.rateLimiter.Wait(ctx); err != nil {
			return
		}

		// Increment operation counter
		atomic.AddInt64(&r.opsCounter, 1)

		// Get next operation
		op := r.scheduler.Next()

		// Execute operation with timeout
		opCtx, cancel := context.WithTimeout(ctx, r.cfg.OpTimeout)
		r.executeOp(opCtx, op, rng)
		cancel()
	}
}

// executeOp executes a single operation
func (r *Runner) executeOp(ctx context.Context, op workload.OpType, rng *rand.Rand) {
	keySeq := r.scheduler.NextKey()
	key := r.keygen.Generate(keySeq)

	var err error

	switch op {
	case workload.OpPut:
		err = r.executePut(ctx, key)
	case workload.OpGet:
		err = r.executeGet(ctx, key, rng)
	case workload.OpDelete:
		err = r.executeDelete(ctx, key)
	case workload.OpCopy:
		err = r.executeCopy(ctx, key, rng)
	case workload.OpList:
		err = r.executeList(ctx)
	case workload.OpHead:
		err = r.executeHead(ctx, key)
	}

	if err != nil {
		r.logger.Debug("operation failed",
			zap.String("op", string(op)),
			zap.String("key", key),
			zap.Error(err),
		)
	}
}

// executePut executes a PUT operation
func (r *Runner) executePut(ctx context.Context, key string) error {
	size := r.sizeDist.Next()

	// Generate data and hash
	reader, hash, err := r.generator.GenerateAndHash(key, size)
	if err != nil {
		return fmt.Errorf("failed to generate data: %w", err)
	}

	// Prepare metadata
	metadata := data.PrepareMetadata(hash, r.cfg.NamespaceTag)

	// Upload with retry
	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err = s3.WithRetry(ctx, retryCfg, r.logger, "put", func(ctx context.Context) error {
		// Reset reader
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to reset reader: %w", err)
		}
		return r.s3Client.PutObject(ctx, key, reader, size, metadata)
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpPut))
	}

	return err
}

// executeGet executes a GET operation
func (r *Runner) executeGet(ctx context.Context, key string, rng *rand.Rand) error {
	shouldVerify := workload.ShouldVerify(r.cfg.VerifyRate, rng)

	var body io.ReadCloser
	var metadata map[string]string
	var err error

	// Download with retry
	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err = s3.WithRetry(ctx, retryCfg, r.logger, "get", func(ctx context.Context) error {
		body, metadata, _, err = r.s3Client.GetObject(ctx, key)
		return err
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpGet))
		return err
	}

	defer body.Close()

	// Verify if requested
	if shouldVerify {
		if err := r.verifier.VerifyWithMetadata(body, metadata); err != nil {
			r.metrics.RecordVerifyFailure()
			r.logger.Warn("verification failed",
				zap.String("key", key),
				zap.Error(err),
			)
			return fmt.Errorf("verification failed: %w", err)
		}
		r.metrics.RecordVerifySuccess()
	} else {
		// Discard body
		io.Copy(io.Discard, body)
	}

	return nil
}

// executeDelete executes a DELETE operation
func (r *Runner) executeDelete(ctx context.Context, key string) error {
	if r.cfg.KeepData {
		// Skip delete in keep-data mode
		return nil
	}

	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err := s3.WithRetry(ctx, retryCfg, r.logger, "delete", func(ctx context.Context) error {
		return r.s3Client.DeleteObject(ctx, key)
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpDelete))
	}

	return err
}

// executeCopy executes a COPY operation
func (r *Runner) executeCopy(ctx context.Context, srcKey string, rng *rand.Rand) error {
	// Generate destination key
	dstSeq := r.scheduler.NextKey()
	dstKey := r.keygen.Generate(dstSeq)

	dstBucket := r.cfg.CopyDstBucket

	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err := s3.WithRetry(ctx, retryCfg, r.logger, "copy", func(ctx context.Context) error {
		return r.s3Client.CopyObject(ctx, srcKey, dstKey, dstBucket)
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpCopy))
	}

	return err
}

// executeList executes a LIST operation
func (r *Runner) executeList(ctx context.Context) error {
	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err := s3.WithRetry(ctx, retryCfg, r.logger, "list", func(ctx context.Context) error {
		_, err := r.s3Client.ListObjects(ctx, r.cfg.Prefix, 1000)
		return err
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpList))
	}

	return err
}

// executeHead executes a HEAD operation
func (r *Runner) executeHead(ctx context.Context, key string) error {
	retryCfg := s3.DefaultRetryConfig()
	retryCfg.MaxAttempts = r.cfg.MaxRetries

	err := s3.WithRetry(ctx, retryCfg, r.logger, "head", func(ctx context.Context) error {
		_, _, err := r.s3Client.HeadObject(ctx, key)
		return err
	})

	if err != nil {
		r.metrics.RecordRetry(string(workload.OpHead))
	}

	return err
}

// runCleanup runs cleanup mode
func (r *Runner) runCleanup(ctx context.Context) error {
	r.logger.Info("running cleanup mode", zap.String("prefix", r.cfg.Prefix))

	deleted, err := r.s3Client.DeleteObjectsByMetadata(
		ctx,
		r.cfg.Prefix,
		data.MetadataKeyCreatedBy,
		data.MetadataValueCreatedBy,
	)

	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	r.logger.Info("cleanup completed", zap.Int("deleted", deleted))
	return nil
}
