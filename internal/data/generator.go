package data

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// Generator generates deterministic or fixed-pattern data
type Generator struct {
	pattern string
	seed    int64
	fixed   []byte
	mu      sync.Mutex
}

// NewGenerator creates a new data generator based on the pattern
// Supported patterns:
//   - "random:<seed>" - deterministic pseudo-random data
//   - "fixed:<hex>" - repeating fixed bytes
func NewGenerator(pattern string) (*Generator, error) {
	g := &Generator{pattern: pattern}

	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid pattern format, expected 'type:value'")
	}

	switch parts[0] {
	case "random":
		seed, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid random seed: %w", err)
		}
		g.seed = seed

	case "fixed":
		data, err := hex.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid fixed hex data: %w", err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("fixed data cannot be empty")
		}
		g.fixed = data

	default:
		return nil, fmt.Errorf("unknown pattern type: %s", parts[0])
	}

	return g, nil
}

// Generate generates data for a specific key and size
// The data is deterministic based on the key and pattern
func (g *Generator) Generate(key string, size int64) io.ReadSeeker {
	switch {
	case g.seed != 0:
		return newRandomReader(key, g.seed, size)
	case len(g.fixed) > 0:
		return newFixedReader(g.fixed, size)
	default:
		// Should not reach here if NewGenerator validated properly
		return newFixedReader([]byte{0}, size)
	}
}

// GenerateAndHash generates data and computes its SHA-256 hash
func (g *Generator) GenerateAndHash(key string, size int64) (io.ReadSeeker, string, error) {
	reader := g.Generate(key, size)

	// Compute hash
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return nil, "", fmt.Errorf("failed to compute hash: %w", err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))

	// Reset reader to beginning
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, "", fmt.Errorf("failed to reset reader: %w", err)
	}

	return reader, hashStr, nil
}

// randomReader implements a deterministic pseudo-random reader
type randomReader struct {
	seed     int64
	rng      *rand.Rand
	size     int64
	position int64
	buf      []byte
}

func newRandomReader(key string, baseSeed int64, size int64) *randomReader {
	// Mix key into seed for per-key determinism
	h := sha256.Sum256([]byte(key))
	keySeed := int64(h[0]) | int64(h[1])<<8 | int64(h[2])<<16 | int64(h[3])<<24
	seed := baseSeed ^ keySeed

	return &randomReader{
		seed: seed,
		rng:  rand.New(rand.NewSource(seed)),
		size: size,
		buf:  make([]byte, 8192), // 8KB buffer
	}
}

func (r *randomReader) Read(p []byte) (n int, err error) {
	if r.position >= r.size {
		return 0, io.EOF
	}

	remaining := r.size - r.position
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	// Fill p with random data
	n = int(toRead)
	for i := 0; i < n; {
		chunk := n - i
		if chunk > len(r.buf) {
			chunk = len(r.buf)
		}
		r.rng.Read(r.buf[:chunk])
		copy(p[i:], r.buf[:chunk])
		i += chunk
	}

	r.position += int64(n)
	return n, nil
}

func (r *randomReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.position + offset
	case io.SeekEnd:
		newPos = r.size + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	if newPos > r.size {
		newPos = r.size
	}

	// Reset RNG if seeking to start
	if newPos == 0 {
		r.rng = rand.New(rand.NewSource(r.seed))
	} else if newPos < r.position {
		// Seeking backwards - need to reset and fast-forward
		r.rng = rand.New(rand.NewSource(r.seed))
		r.position = 0

		// Fast-forward to new position
		if newPos > 0 {
			_, err := io.CopyN(io.Discard, r, newPos)
			if err != nil && err != io.EOF {
				return 0, err
			}
		}
		return r.position, nil
	}
	// For forward seeks, just update position (data will be generated on next read)

	r.position = newPos
	return r.position, nil
}

// fixedReader implements a repeating fixed-pattern reader
type fixedReader struct {
	pattern  []byte
	size     int64
	position int64
}

func newFixedReader(pattern []byte, size int64) *fixedReader {
	return &fixedReader{
		pattern: pattern,
		size:    size,
	}
}

func (r *fixedReader) Read(p []byte) (n int, err error) {
	if r.position >= r.size {
		return 0, io.EOF
	}

	remaining := r.size - r.position
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	n = int(toRead)
	patternLen := int64(len(r.pattern))

	for i := 0; i < n; i++ {
		offset := (r.position + int64(i)) % patternLen
		p[i] = r.pattern[offset]
	}

	r.position += int64(n)
	return n, nil
}

func (r *fixedReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.position + offset
	case io.SeekEnd:
		newPos = r.size + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	if newPos > r.size {
		return 0, fmt.Errorf("position beyond size")
	}

	r.position = newPos
	return r.position, nil
}

// SizeDistribution generates object sizes based on a distribution
type SizeDistribution interface {
	Next() int64
}

// FixedSize always returns the same size
type FixedSize struct {
	size int64
}

func NewFixedSize(size int64) *FixedSize {
	return &FixedSize{size: size}
}

func (f *FixedSize) Next() int64 {
	return f.size
}

// LogNormalSize generates sizes from a log-normal distribution
type LogNormalSize struct {
	mean   float64
	stddev float64
	rng    *rand.Rand
	mu     sync.Mutex
}

func NewLogNormalSize(mean, stddev float64, seed int64) *LogNormalSize {
	return &LogNormalSize{
		mean:   mean,
		stddev: stddev,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

func (l *LogNormalSize) Next() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Generate log-normal random variable
	// log(X) ~ N(mu, sigma^2)
	// We want mean and stddev in linear space, so convert
	mu := math.Log(l.mean) - 0.5*math.Log(1+l.stddev*l.stddev/(l.mean*l.mean))
	sigma := math.Sqrt(math.Log(1 + l.stddev*l.stddev/(l.mean*l.mean)))

	logValue := l.rng.NormFloat64()*sigma + mu
	size := int64(math.Exp(logValue))

	// Ensure minimum size of 1 byte
	if size < 1 {
		size = 1
	}

	return size
}

// UniformSize generates sizes uniformly between min and max
type UniformSize struct {
	min int64
	max int64
	rng *rand.Rand
	mu  sync.Mutex
}

func NewUniformSize(min, max int64, seed int64) *UniformSize {
	return &UniformSize{
		min: min,
		max: max,
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (u *UniformSize) Next() int64 {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.min == u.max {
		return u.min
	}

	return u.min + u.rng.Int63n(u.max-u.min+1)
}

// ParseSize parses a size string like "1MiB", "512KB", "1.5GB"
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Find where the number ends
	var numStr string
	var unitStr string

	foundUnit := false
	for i, c := range s {
		if (c < '0' || c > '9') && c != '.' {
			numStr = s[:i]
			unitStr = s[i:]
			foundUnit = true
			break
		}
	}

	if !foundUnit {
		// No unit found, entire string is number
		numStr = s
		unitStr = ""
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in size string")
	}

	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	// Parse unit
	unitStr = strings.TrimSpace(unitStr)
	var multiplier int64

	switch strings.ToUpper(unitStr) {
	case "", "B":
		multiplier = 1
	case "KB", "K":
		multiplier = 1000
	case "KIB", "KI":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1000 * 1000
	case "MIB", "MI":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1000 * 1000 * 1000
	case "GIB", "GI":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}

	return int64(value * float64(multiplier)), nil
}

// ParseSizeDistribution parses a size distribution string
// Examples:
//   - "fixed:1MiB"
//   - "dist:lognormal:mean=1MiB,std=0.5"
//   - "uniform:min=1KB,max=10MB"
func ParseSizeDistribution(s string, seed int64) (SizeDistribution, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid size distribution format")
	}

	switch parts[0] {
	case "fixed":
		size, err := ParseSize(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid fixed size: %w", err)
		}
		return NewFixedSize(size), nil

	case "dist":
		// Parse dist:lognormal:mean=1MiB,std=0.5
		subParts := strings.SplitN(parts[1], ":", 2)
		if len(subParts) != 2 {
			return nil, fmt.Errorf("invalid distribution format")
		}

		distType := subParts[0]
		params := parseParams(subParts[1])

		switch distType {
		case "lognormal":
			meanStr, ok := params["mean"]
			if !ok {
				return nil, fmt.Errorf("lognormal distribution requires 'mean' parameter")
			}
			mean, err := ParseSize(meanStr)
			if err != nil {
				return nil, fmt.Errorf("invalid mean: %w", err)
			}

			stdStr, ok := params["std"]
			if !ok {
				stdStr = "0.5" // default
			}
			std, err := strconv.ParseFloat(stdStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid std: %w", err)
			}

			return NewLogNormalSize(float64(mean), std*float64(mean), seed), nil

		default:
			return nil, fmt.Errorf("unknown distribution type: %s", distType)
		}

	case "uniform":
		params := parseParams(parts[1])

		minStr, ok := params["min"]
		if !ok {
			return nil, fmt.Errorf("uniform distribution requires 'min' parameter")
		}
		min, err := ParseSize(minStr)
		if err != nil {
			return nil, fmt.Errorf("invalid min: %w", err)
		}

		maxStr, ok := params["max"]
		if !ok {
			return nil, fmt.Errorf("uniform distribution requires 'max' parameter")
		}
		max, err := ParseSize(maxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid max: %w", err)
		}

		return NewUniformSize(min, max, seed), nil

	default:
		return nil, fmt.Errorf("unknown size type: %s", parts[0])
	}
}

func parseParams(s string) map[string]string {
	params := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return params
}
