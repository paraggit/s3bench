package workload

import (
	"math/rand"
	"testing"
)

func TestNewScheduler(t *testing.T) {
	tests := []struct {
		name    string
		mix     map[string]int
		wantErr bool
	}{
		{"valid mix", map[string]int{"put": 50, "get": 50}, false},
		{"empty mix", map[string]int{}, true},
		{"zero sum", map[string]int{"put": 0, "get": 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScheduler(tt.mix, 1000, 42)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewScheduler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchedulerNext(t *testing.T) {
	mix := map[string]int{
		"put": 50,
		"get": 50,
	}

	scheduler, err := NewScheduler(mix, 1000, 42)
	if err != nil {
		t.Fatalf("NewScheduler() failed: %v", err)
	}

	// Generate many operations
	counts := make(map[OpType]int)
	samples := 10000

	for i := 0; i < samples; i++ {
		op := scheduler.Next()
		counts[op]++
	}

	// Check distribution is roughly 50/50 (allow 10% deviation)
	putPct := float64(counts[OpPut]) / float64(samples) * 100
	getPct := float64(counts[OpGet]) / float64(samples) * 100

	if putPct < 40 || putPct > 60 {
		t.Errorf("PUT percentage = %.1f%%, want ~50%%", putPct)
	}

	if getPct < 40 || getPct > 60 {
		t.Errorf("GET percentage = %.1f%%, want ~50%%", getPct)
	}

	t.Logf("Distribution: PUT=%.1f%%, GET=%.1f%%", putPct, getPct)
}

func TestSchedulerNextKey(t *testing.T) {
	mix := map[string]int{"get": 100}
	totalKeys := 100

	scheduler, err := NewScheduler(mix, totalKeys, 42)
	if err != nil {
		t.Fatalf("NewScheduler() failed: %v", err)
	}

	// Generate many keys
	for i := 0; i < 1000; i++ {
		key := scheduler.NextKey()
		if key < 0 || key >= totalKeys {
			t.Errorf("NextKey() = %d, want 0 <= key < %d", key, totalKeys)
		}
	}
}

func TestShouldVerify(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := []struct {
		name       string
		verifyRate float64
		samples    int
		wantMin    float64
		wantMax    float64
	}{
		{"never", 0.0, 1000, 0.0, 0.0},
		{"always", 1.0, 1000, 1.0, 1.0},
		{"half", 0.5, 10000, 0.45, 0.55},
		{"tenth", 0.1, 10000, 0.05, 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			for i := 0; i < tt.samples; i++ {
				if ShouldVerify(tt.verifyRate, rng) {
					count++
				}
			}

			rate := float64(count) / float64(tt.samples)
			if rate < tt.wantMin || rate > tt.wantMax {
				t.Errorf("ShouldVerify() rate = %.3f, want %.3f-%.3f", rate, tt.wantMin, tt.wantMax)
			} else {
				t.Logf("ShouldVerify() rate = %.3f (expected ~%.3f)", rate, tt.verifyRate)
			}
		})
	}
}
