package data

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	// MetadataKeySHA256 is the key for storing SHA-256 hash in object metadata
	MetadataKeySHA256 = "sha256"

	// MetadataKeyCreatedBy marks objects created by this tool
	MetadataKeyCreatedBy = "created-by"

	// MetadataValueCreatedBy is the value for created-by metadata
	MetadataValueCreatedBy = "s3-workload"
)

// Verifier verifies data integrity
type Verifier struct {
	generator *Generator
}

// NewVerifier creates a new verifier
func NewVerifier(generator *Generator) *Verifier {
	return &Verifier{generator: generator}
}

// ComputeHash computes SHA-256 hash of a reader
func ComputeHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Verify verifies that the data from reader matches the expected hash
func (v *Verifier) Verify(r io.Reader, expectedHash string) error {
	actualHash, err := ComputeHash(r)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// VerifyKey verifies that the data from reader matches the expected data for a given key
func (v *Verifier) VerifyKey(r io.Reader, key string, size int64) error {
	// Generate expected data
	expectedReader := v.generator.Generate(key, size)
	expectedHash, err := ComputeHash(expectedReader)
	if err != nil {
		return fmt.Errorf("failed to compute expected hash: %w", err)
	}

	// Verify actual data
	return v.Verify(r, expectedHash)
}

// VerifyWithMetadata verifies data using hash from metadata
func (v *Verifier) VerifyWithMetadata(r io.Reader, metadata map[string]string) error {
	expectedHash, ok := metadata[MetadataKeySHA256]
	if !ok {
		return fmt.Errorf("no hash found in metadata")
	}

	return v.Verify(r, expectedHash)
}

// PrepareMetadata prepares object metadata with hash and tracking info
func PrepareMetadata(hash string, namespaceTag string) map[string]string {
	metadata := map[string]string{
		MetadataKeySHA256:    hash,
		MetadataKeyCreatedBy: MetadataValueCreatedBy,
	}

	// Add namespace tag if provided (e.g., "env=perf")
	if namespaceTag != "" {
		parts := splitTag(namespaceTag)
		for k, v := range parts {
			metadata[k] = v
		}
	}

	return metadata
}

func splitTag(tag string) map[string]string {
	result := make(map[string]string)
	for _, pair := range splitCommaSeparated(tag) {
		kv := splitEqual(pair)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

func splitCommaSeparated(s string) []string {
	var result []string
	for _, part := range splitString(s, ',') {
		if trimmed := trim(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitEqual(s string) []string {
	var result []string
	for _, part := range splitString(s, '=') {
		result = append(result, trim(part))
	}
	return result
}

func splitString(s string, sep rune) []string {
	var result []string
	var current string

	for _, c := range s {
		if c == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}

	if current != "" || len(result) == 0 {
		result = append(result, current)
	}

	return result
}

func trim(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}
