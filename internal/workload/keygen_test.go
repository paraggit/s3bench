package workload

import (
	"testing"
)

func TestKeyGenerator(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		template string
		seq      int
		want     string
	}{
		{
			name:     "simple seq",
			prefix:   "test/",
			template: "obj-{seq}.bin",
			seq:      42,
			want:     "test/obj-42.bin",
		},
		{
			name:     "padded seq",
			prefix:   "bench/",
			template: "obj-{seq:08}.bin",
			seq:      42,
			want:     "bench/obj-00000042.bin",
		},
		{
			name:     "no placeholder",
			prefix:   "data/",
			template: "fixed.bin",
			seq:      42,
			want:     "data/fixed.bin",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			template: "obj-{seq:06}.dat",
			seq:      123,
			want:     "obj-000123.dat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kg := NewKeyGenerator(tt.prefix, tt.template, 1000)
			got := kg.Generate(tt.seq)
			if got != tt.want {
				t.Errorf("Generate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKeyGeneratorCount(t *testing.T) {
	kg := NewKeyGenerator("prefix/", "obj-{seq}.bin", 12345)
	if kg.Count() != 12345 {
		t.Errorf("Count() = %d, want 12345", kg.Count())
	}
}
