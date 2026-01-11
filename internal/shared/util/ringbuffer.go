package util

// RingBuffer is a fixed-size circular buffer that overwrites oldest elements when full.
// It provides O(1) push operations with zero allocations after initialization.
type RingBuffer[T any] struct {
	data  []T
	head  int // Next write position
	count int // Number of valid elements
	size  int // Maximum capacity
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer[T]{
		data: make([]T, capacity),
		size: capacity,
	}
}

// Push adds a value to the buffer, overwriting the oldest value if full.
func (r *RingBuffer[T]) Push(value T) {
	r.data[r.head] = value
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
}

// Len returns the number of elements in the buffer.
func (r *RingBuffer[T]) Len() int {
	return r.count
}

// Cap returns the capacity of the buffer.
func (r *RingBuffer[T]) Cap() int {
	return r.size
}

// Get returns the element at the specified index (0 = oldest, Len()-1 = newest).
// Returns zero value if index is out of bounds.
func (r *RingBuffer[T]) Get(index int) T {
	var zero T
	if index < 0 || index >= r.count {
		return zero
	}
	// Calculate actual position in circular buffer
	// Start from oldest element
	start := (r.head - r.count + r.size) % r.size
	pos := (start + index) % r.size
	return r.data[pos]
}

// ToSlice returns a copy of all elements in order (oldest to newest).
// This allocates a new slice on every call.
//
// Performance note: ToSlice is called during Render() for graph-mode widgets
// (every 100ms). Benchmark analysis (see ringbuffer_bench_test.go) shows:
//
//   - Single call: ~200ns, 512B allocation (64 float64 elements)
//   - Dual I/O render: ~400ns, 1KB allocation (2 calls)
//   - Full render with normalization: ~770ns, 2KB (4 allocations total)
//
// This represents 0.0008% of the 100ms render cycle - negligible.
// A zero-allocation CopyTo() approach was benchmarked at ~270ns but requires:
//   - Widgets to manage pre-allocated buffers
//   - Buffer lifecycle management (resizing if history length changes)
//   - Additional complexity in multiple widget implementations
//
// The current approach prioritizes simplicity and correctness over
// micro-optimization. The GC easily handles 60KB/sec of short-lived allocations.
func (r *RingBuffer[T]) ToSlice() []T {
	if r.count == 0 {
		return nil
	}
	result := make([]T, r.count)
	start := (r.head - r.count + r.size) % r.size
	for i := 0; i < r.count; i++ {
		result[i] = r.data[(start+i)%r.size]
	}
	return result
}

// Clear removes all elements from the buffer.
func (r *RingBuffer[T]) Clear() {
	r.head = 0
	r.count = 0
}

// IsFull returns true if the buffer is at capacity.
func (r *RingBuffer[T]) IsFull() bool {
	return r.count == r.size
}

// IsEmpty returns true if the buffer contains no elements.
func (r *RingBuffer[T]) IsEmpty() bool {
	return r.count == 0
}
