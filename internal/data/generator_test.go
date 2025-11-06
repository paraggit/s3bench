package data

import (
	"io"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"random pattern", "random:42", false},
		{"fixed pattern", "fixed:DEADBEEF", false},
		{"invalid format", "invalid", true},
		{"invalid seed", "random:notanumber", true},
		{"invalid hex", "fixed:GGGG", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGenerator(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGenerator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGeneratorDeterminism(t *testing.T) {
	gen, err := NewGenerator("random:42")
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	key := "test-key"
	size := int64(1024)

	// Generate data twice
	reader1 := gen.Generate(key, size)
	data1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("failed to read data1: %v", err)
	}

	reader2 := gen.Generate(key, size)
	data2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("failed to read data2: %v", err)
	}

	// Should be identical
	if len(data1) != len(data2) {
		t.Errorf("data length mismatch: %d != %d", len(data1), len(data2))
	}

	for i := range data1 {
		if data1[i] != data2[i] {
			t.Errorf("data mismatch at position %d", i)
			break
		}
	}
}

func TestGenerateAndHash(t *testing.T) {
	gen, err := NewGenerator("random:42")
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	key := "test-key"
	size := int64(1024)

	reader, hash, err := gen.GenerateAndHash(key, size)
	if err != nil {
		t.Fatalf("failed to generate and hash: %v", err)
	}

	if hash == "" {
		t.Error("hash is empty")
	}

	if len(hash) != 64 { // SHA-256 hex is 64 characters
		t.Errorf("hash length is %d, expected 64", len(hash))
	}

	// Verify reader is at start
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read data: %v", err)
	}

	if int64(len(data)) != size {
		t.Errorf("data size is %d, expected %d", len(data), size)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"bytes", "100", 100, false},
		{"kilobytes", "1KB", 1000, false},
		{"kibibytes", "1KiB", 1024, false},
		{"megabytes", "1MB", 1000000, false},
		{"mebibytes", "1MiB", 1048576, false},
		{"gigabytes", "1GB", 1000000000, false},
		{"gibibytes", "1GiB", 1073741824, false},
		{"decimal", "1.5MB", 1500000, false},
		{"invalid", "invalid", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSizeDistribution(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"fixed size", "fixed:1MiB", false},
		{"lognormal dist", "dist:lognormal:mean=1MiB,std=0.5", false},
		{"uniform dist", "uniform:min=1KB,max=10MB", false},
		{"invalid format", "invalid", true},
		{"unknown type", "unknown:value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSizeDistribution(tt.input, 42)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSizeDistribution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFixedSizeDistribution(t *testing.T) {
	dist := NewFixedSize(1024)

	for i := 0; i < 10; i++ {
		size := dist.Next()
		if size != 1024 {
			t.Errorf("FixedSize.Next() = %d, want 1024", size)
		}
	}
}

func TestLogNormalSizeDistribution(t *testing.T) {
	dist := NewLogNormalSize(1024, 512, 42)

	// Generate some samples
	var sum int64
	samples := 100
	for i := 0; i < samples; i++ {
		size := dist.Next()
		if size < 1 {
			t.Errorf("LogNormalSize.Next() = %d, should be >= 1", size)
		}
		sum += size
	}

	// Check average is reasonable (within 50% of mean)
	avg := sum / int64(samples)
	if avg < 512 || avg > 2048 {
		t.Logf("Warning: average size %d is far from mean 1024 (acceptable for small sample)", avg)
	}
}
