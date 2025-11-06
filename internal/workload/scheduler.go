package workload

import (
	"fmt"
	"math/rand"
	"sync"
)

// OpType represents an operation type
type OpType string

const (
	OpPut          OpType = "put"
	OpGet          OpType = "get"
	OpDelete       OpType = "delete"
	OpCopy         OpType = "copy"
	OpList         OpType = "list"
	OpHead         OpType = "head"
	OpMultipartPut OpType = "multipart_put"
)

// Scheduler schedules operations based on the configured mix
type Scheduler struct {
	mix       map[OpType]int // percentage for each op
	weights   []int          // cumulative weights
	ops       []OpType       // operations in order
	totalKeys int
	rng       *rand.Rand
	mu        sync.Mutex
}

// NewScheduler creates a new operation scheduler
func NewScheduler(mix map[string]int, totalKeys int, seed int64) (*Scheduler, error) {
	if len(mix) == 0 {
		return nil, fmt.Errorf("operation mix cannot be empty")
	}

	// Normalize mix to 100
	total := 0
	for _, pct := range mix {
		total += pct
	}
	if total == 0 {
		return nil, fmt.Errorf("operation mix percentages sum to zero")
	}

	s := &Scheduler{
		mix:       make(map[OpType]int),
		totalKeys: totalKeys,
		rng:       rand.New(rand.NewSource(seed)),
	}

	// Convert string keys to OpType and build cumulative weights
	var cumulative int
	for opStr, pct := range mix {
		if pct == 0 {
			continue
		}

		op := OpType(opStr)
		s.mix[op] = pct
		s.ops = append(s.ops, op)
		cumulative += pct
		s.weights = append(s.weights, cumulative)
	}

	return s, nil
}

// Next returns the next operation to execute
func (s *Scheduler) Next() OpType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.ops) == 0 {
		return OpGet // fallback
	}

	if len(s.ops) == 1 {
		return s.ops[0]
	}

	// Weighted random selection
	r := s.rng.Intn(s.weights[len(s.weights)-1])

	for i, weight := range s.weights {
		if r < weight {
			return s.ops[i]
		}
	}

	return s.ops[len(s.ops)-1]
}

// NextKey returns a random key from the keyspace
func (s *Scheduler) NextKey() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rng.Intn(s.totalKeys)
}

// ShouldVerify returns true if verification should be performed based on verify rate
func ShouldVerify(verifyRate float64, rng *rand.Rand) bool {
	if verifyRate <= 0 {
		return false
	}
	if verifyRate >= 1.0 {
		return true
	}
	return rng.Float64() < verifyRate
}
