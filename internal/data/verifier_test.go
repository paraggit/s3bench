package data

import (
	"bytes"
	"testing"
)

func TestComputeHash(t *testing.T) {
	data := []byte("hello world")
	hash, err := ComputeHash(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ComputeHash() failed: %v", err)
	}

	// SHA-256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("ComputeHash() = %s, want %s", hash, expected)
	}
}

func TestVerify(t *testing.T) {
	gen, err := NewGenerator("random:42")
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier := NewVerifier(gen)

	key := "test-key"
	size := int64(1024)

	// Generate data
	reader, hash, err := gen.GenerateAndHash(key, size)
	if err != nil {
		t.Fatalf("failed to generate data: %v", err)
	}

	// Verify should succeed
	if err := verifier.Verify(reader, hash); err != nil {
		t.Errorf("Verify() failed: %v", err)
	}
}

func TestVerifyMismatch(t *testing.T) {
	gen, err := NewGenerator("random:42")
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier := NewVerifier(gen)

	data := []byte("test data")
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	// Verify should fail
	if err := verifier.Verify(bytes.NewReader(data), wrongHash); err == nil {
		t.Error("Verify() should have failed with wrong hash")
	}
}

func TestPrepareMetadata(t *testing.T) {
	hash := "abcd1234"
	namespaceTag := "env=test,team=platform"

	metadata := PrepareMetadata(hash, namespaceTag)

	if metadata[MetadataKeySHA256] != hash {
		t.Errorf("metadata hash = %s, want %s", metadata[MetadataKeySHA256], hash)
	}

	if metadata[MetadataKeyCreatedBy] != MetadataValueCreatedBy {
		t.Errorf("metadata created-by = %s, want %s", metadata[MetadataKeyCreatedBy], MetadataValueCreatedBy)
	}

	if metadata["env"] != "test" {
		t.Errorf("metadata env = %s, want test", metadata["env"])
	}

	if metadata["team"] != "platform" {
		t.Errorf("metadata team = %s, want platform", metadata["team"])
	}
}
