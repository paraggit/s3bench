package workload

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyGenerator generates object keys
type KeyGenerator struct {
	prefix   string
	template string
	keys     int
}

// NewKeyGenerator creates a new key generator
func NewKeyGenerator(prefix, template string, keys int) *KeyGenerator {
	return &KeyGenerator{
		prefix:   prefix,
		template: template,
		keys:     keys,
	}
}

// Generate generates a key for a given sequence number
func (kg *KeyGenerator) Generate(seq int) string {
	key := kg.template

	// Replace {seq} or {seq:format} placeholders
	// Example: obj-{seq:08}.bin -> obj-00000042.bin
	key = replacePlaceholder(key, seq)

	return kg.prefix + key
}

// Count returns the total number of keys
func (kg *KeyGenerator) Count() int {
	return kg.keys
}

func replacePlaceholder(template string, seq int) string {
	// Find {seq...} pattern
	start := -1
	for i := 0; i < len(template); i++ {
		if i+4 <= len(template) && template[i:i+4] == "{seq" {
			start = i
			break
		}
	}

	if start == -1 {
		return template
	}

	// Find closing brace
	end := -1
	for i := start; i < len(template); i++ {
		if template[i] == '}' {
			end = i
			break
		}
	}

	if end == -1 {
		return template
	}

	// Extract format if present
	placeholder := template[start : end+1]
	var formatted string

	if placeholder == "{seq}" {
		formatted = strconv.Itoa(seq)
	} else if strings.HasPrefix(placeholder, "{seq:") {
		// Extract format like "08" from "{seq:08}"
		format := placeholder[5 : len(placeholder)-1]
		if width, err := strconv.Atoi(format); err == nil && width > 0 {
			formatted = fmt.Sprintf("%0*d", width, seq)
		} else {
			formatted = strconv.Itoa(seq)
		}
	} else {
		formatted = strconv.Itoa(seq)
	}

	return template[:start] + formatted + template[end+1:]
}
