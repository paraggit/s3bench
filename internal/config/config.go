package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds all configuration for the s3-workload tool
type Config struct {
	// S3 Connection
	Endpoint      string `mapstructure:"endpoint"`
	Region        string `mapstructure:"region"`
	Bucket        string `mapstructure:"bucket"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	PathStyle     bool   `mapstructure:"path_style"`
	SkipTLSVerify bool   `mapstructure:"skip_tls_verify"`

	// Bucket Management
	CreateBucket bool   `mapstructure:"create_bucket"`
	Versioning   string `mapstructure:"versioning"` // "on", "off", "keep"

	// Workload Parameters
	Concurrency int            `mapstructure:"concurrency"`
	Mix         map[string]int `mapstructure:"mix"` // op -> percentage
	Duration    time.Duration  `mapstructure:"duration"`
	Operations  int64          `mapstructure:"operations"`

	// Object Configuration
	Size        string `mapstructure:"size"` // "fixed:1MiB", "dist:lognormal:mean=1MiB,std=0.6"
	Keys        int    `mapstructure:"keys"`
	Prefix      string `mapstructure:"prefix"`
	KeyTemplate string `mapstructure:"key_template"`
	RandomKeys  bool   `mapstructure:"random_keys"`

	// Data Pattern & Verification
	Pattern    string  `mapstructure:"pattern"`     // "random:42", "fixed:DEADBEEF"
	VerifyRate float64 `mapstructure:"verify_rate"` // 0.0 - 1.0

	// Rate Limiting
	RateType  string  `mapstructure:"rate_type"`  // "fixed", "poisson"
	RateLimit float64 `mapstructure:"rate_limit"` // QPS for fixed, lambda for poisson

	// Timeouts & Retries
	OpTimeout    time.Duration `mapstructure:"op_timeout"`
	MaxRetries   int           `mapstructure:"max_retries"`
	RetryBackoff time.Duration `mapstructure:"retry_backoff"`

	// Copy Operation
	CopyDstBucket string `mapstructure:"copy_dst_bucket"`

	// Safety & Cleanup
	NamespaceTag string `mapstructure:"namespace_tag"`
	KeepData     bool   `mapstructure:"keep_data"`
	Cleanup      bool   `mapstructure:"cleanup"`
	DryRun       bool   `mapstructure:"dry_run"`

	// Observability
	MetricsPort int    `mapstructure:"metrics_port"`
	HTTPBind    string `mapstructure:"http_bind"`
	LogLevel    string `mapstructure:"log_level"`
	PprofPort   int    `mapstructure:"pprof_port"`

	// Internal
	ConfigFile string `mapstructure:"config"`
}

// NewConfig returns a Config with sensible defaults
func NewConfig() *Config {
	return &Config{
		Region:        "us-east-1",
		PathStyle:     false,
		SkipTLSVerify: false,
		Versioning:    "keep",

		Concurrency: 32,
		Mix:         map[string]int{"put": 50, "get": 50},
		Duration:    10 * time.Minute,
		Operations:  0, // unlimited unless set

		Size:        "fixed:1MiB",
		Keys:        10000,
		Prefix:      "",
		KeyTemplate: "obj-{seq:08}.bin",
		RandomKeys:  false,

		Pattern:    "random:42",
		VerifyRate: 0.1,

		RateType:  "fixed",
		RateLimit: 0, // unlimited

		OpTimeout:    30 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,

		KeepData: false,
		Cleanup:  false,
		DryRun:   false,

		MetricsPort: 9090,
		HTTPBind:    "0.0.0.0",
		LogLevel:    "info",
		PprofPort:   0, // disabled
	}
}

// BindFlags binds pflag flags to viper
func (c *Config) BindFlags(flags *pflag.FlagSet) error {
	// S3 Connection
	flags.String("endpoint", c.Endpoint, "S3 endpoint URL")
	flags.String("region", c.Region, "AWS region")
	flags.String("bucket", c.Bucket, "S3 bucket name")
	flags.String("access-key", c.AccessKey, "AWS access key (or use AWS_ACCESS_KEY_ID env)")
	flags.String("secret-key", c.SecretKey, "AWS secret key (or use AWS_SECRET_ACCESS_KEY env)")
	flags.Bool("path-style", c.PathStyle, "Use path-style addressing")
	flags.Bool("skip-tls-verify", c.SkipTLSVerify, "Skip TLS certificate verification")

	// Bucket Management
	flags.Bool("create-bucket", c.CreateBucket, "Create bucket if it doesn't exist")
	flags.String("versioning", c.Versioning, "Bucket versioning: on, off, keep")

	// Workload Parameters
	flags.Int("concurrency", c.Concurrency, "Number of concurrent workers")
	flags.StringToInt("mix", c.Mix, "Operation mix (e.g., put=40,get=40,delete=10,copy=5,list=5)")
	flags.Duration("duration", c.Duration, "Workload duration (0 for unlimited)")
	flags.Int64("operations", c.Operations, "Total operations (0 for unlimited)")

	// Object Configuration
	flags.String("size", c.Size, "Object size: fixed:1MiB or dist:lognormal:mean=1MiB,std=0.6")
	flags.Int("keys", c.Keys, "Number of unique keys in keyspace")
	flags.String("prefix", c.Prefix, "Key prefix")
	flags.String("key-template", c.KeyTemplate, "Key template with {seq} placeholder")
	flags.Bool("random-keys", c.RandomKeys, "Use random key selection")

	// Data Pattern & Verification
	flags.String("pattern", c.Pattern, "Data pattern: random:<seed> or fixed:<hex>")
	flags.Float64("verify-rate", c.VerifyRate, "Fraction of GETs to verify (0.0-1.0)")

	// Rate Limiting
	flags.String("rate-type", c.RateType, "Rate limiter type: fixed or poisson")
	flags.Float64("rate-limit", c.RateLimit, "Rate limit (QPS for fixed, lambda for poisson)")

	// Timeouts & Retries
	flags.Duration("op-timeout", c.OpTimeout, "Per-operation timeout")
	flags.Int("max-retries", c.MaxRetries, "Maximum retry attempts")
	flags.Duration("retry-backoff", c.RetryBackoff, "Initial retry backoff")

	// Copy Operation
	flags.String("copy-dst-bucket", c.CopyDstBucket, "Destination bucket for COPY operations")

	// Safety & Cleanup
	flags.String("namespace-tag", c.NamespaceTag, "Namespace tag for object metadata (e.g., env=perf)")
	flags.Bool("keep-data", c.KeepData, "Keep data (skip cleanup deletes)")
	flags.Bool("cleanup", c.Cleanup, "Cleanup mode: delete only tool-created objects")
	flags.Bool("dry-run", c.DryRun, "Dry run: print config and exit")

	// Observability
	flags.Int("metrics-port", c.MetricsPort, "Prometheus metrics port")
	flags.String("http-bind", c.HTTPBind, "HTTP bind address")
	flags.String("log-level", c.LogLevel, "Log level: debug, info, warn, error")
	flags.Int("pprof-port", c.PprofPort, "Pprof port (0 to disable)")

	// Config File
	flags.String("config", c.ConfigFile, "Config file path")

	// Bind all flags to viper
	return viper.BindPFlags(flags)
}

// Load loads configuration from file, flags, and environment
func Load(flags *pflag.FlagSet) (*Config, error) {
	cfg := NewConfig()

	// Bind flags
	if err := cfg.BindFlags(flags); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	// Load config file if specified
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables override
	viper.SetEnvPrefix("S3BENCH")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Load AWS credentials from standard env vars if not set
	if viper.GetString("access_key") == "" {
		if val := os.Getenv("AWS_ACCESS_KEY_ID"); val != "" {
			viper.Set("access_key", val)
		}
	}
	if viper.GetString("secret_key") == "" {
		if val := os.Getenv("AWS_SECRET_ACCESS_KEY"); val != "" {
			viper.Set("secret_key", val)
		}
	}

	// Unmarshal config
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if c.Keys < 1 {
		return fmt.Errorf("keys must be >= 1")
	}
	if c.VerifyRate < 0 || c.VerifyRate > 1 {
		return fmt.Errorf("verify-rate must be between 0.0 and 1.0")
	}

	// Validate operation mix
	if len(c.Mix) == 0 {
		return fmt.Errorf("operation mix cannot be empty")
	}
	total := 0
	for op, pct := range c.Mix {
		if pct < 0 || pct > 100 {
			return fmt.Errorf("operation %s has invalid percentage: %d", op, pct)
		}
		total += pct
	}
	if total == 0 {
		return fmt.Errorf("operation mix percentages sum to zero")
	}

	// Normalize mix to 100%
	if total != 100 {
		factor := 100.0 / float64(total)
		for op := range c.Mix {
			c.Mix[op] = int(float64(c.Mix[op]) * factor)
		}
	}

	// Validate versioning
	if c.Versioning != "on" && c.Versioning != "off" && c.Versioning != "keep" {
		return fmt.Errorf("versioning must be 'on', 'off', or 'keep'")
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// NormalizeMix normalizes operation mix to sum to 100
func (c *Config) NormalizeMix() {
	total := 0
	for _, pct := range c.Mix {
		total += pct
	}
	if total == 0 {
		return
	}

	factor := 100.0 / float64(total)
	normalized := make(map[string]int)
	for op, pct := range c.Mix {
		normalized[op] = int(float64(pct) * factor)
	}
	c.Mix = normalized
}
