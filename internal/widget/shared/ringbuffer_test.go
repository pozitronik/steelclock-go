package shared

import (
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Len() != 0 {
		t.Errorf("Len() = %d, want 0", rb.Len())
	}
	if rb.Cap() != 3 {
		t.Errorf("Cap() = %d, want 3", rb.Cap())
	}
	if !rb.IsEmpty() {
		t.Error("IsEmpty() should be true")
	}

	rb.Push(1)
	rb.Push(2)

	if rb.Len() != 2 {
		t.Errorf("Len() = %d, want 2", rb.Len())
	}
	if rb.IsEmpty() {
		t.Error("IsEmpty() should be false")
	}
	if rb.IsFull() {
		t.Error("IsFull() should be false")
	}

	rb.Push(3)

	if !rb.IsFull() {
		t.Error("IsFull() should be true")
	}
	if rb.Len() != 3 {
		t.Errorf("Len() = %d, want 3", rb.Len())
	}
}

func TestRingBuffer_Overwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // Overwrites 1
	rb.Push(5) // Overwrites 2

	if rb.Len() != 3 {
		t.Errorf("Len() = %d, want 3", rb.Len())
	}

	// Should contain [3, 4, 5] in order (oldest to newest)
	slice := rb.ToSlice()
	expected := []int{3, 4, 5}
	if len(slice) != len(expected) {
		t.Fatalf("ToSlice() len = %d, want %d", len(slice), len(expected))
	}
	for i, v := range expected {
		if slice[i] != v {
			t.Errorf("ToSlice()[%d] = %d, want %d", i, slice[i], v)
		}
	}
}

func TestRingBuffer_Get(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(10)
	rb.Push(20)
	rb.Push(30)

	// Index 0 = oldest, Index 2 = newest
	if rb.Get(0) != 10 {
		t.Errorf("Get(0) = %d, want 10", rb.Get(0))
	}
	if rb.Get(1) != 20 {
		t.Errorf("Get(1) = %d, want 20", rb.Get(1))
	}
	if rb.Get(2) != 30 {
		t.Errorf("Get(2) = %d, want 30", rb.Get(2))
	}

	// Out of bounds should return zero value
	if rb.Get(-1) != 0 {
		t.Errorf("Get(-1) = %d, want 0", rb.Get(-1))
	}
	if rb.Get(3) != 0 {
		t.Errorf("Get(3) = %d, want 0", rb.Get(3))
	}
}

func TestRingBuffer_GetAfterOverwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // Overwrites 1

	// Should now be [2, 3, 4]
	if rb.Get(0) != 2 {
		t.Errorf("Get(0) = %d, want 2", rb.Get(0))
	}
	if rb.Get(1) != 3 {
		t.Errorf("Get(1) = %d, want 3", rb.Get(1))
	}
	if rb.Get(2) != 4 {
		t.Errorf("Get(2) = %d, want 4", rb.Get(2))
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", rb.Len())
	}
	if !rb.IsEmpty() {
		t.Error("IsEmpty() after Clear() should be true")
	}
}

func TestRingBuffer_ToSliceEmpty(t *testing.T) {
	rb := NewRingBuffer[int](3)

	slice := rb.ToSlice()
	if slice != nil {
		t.Errorf("ToSlice() on empty buffer = %v, want nil", slice)
	}
}

func TestRingBuffer_ZeroCapacity(t *testing.T) {
	rb := NewRingBuffer[int](0)

	// Should default to capacity 1
	if rb.Cap() != 1 {
		t.Errorf("Cap() with 0 capacity = %d, want 1", rb.Cap())
	}

	rb.Push(42)
	if rb.Get(0) != 42 {
		t.Errorf("Get(0) = %d, want 42", rb.Get(0))
	}
}

func TestRingBuffer_Float64(t *testing.T) {
	rb := NewRingBuffer[float64](5)

	rb.Push(1.1)
	rb.Push(2.2)
	rb.Push(3.3)

	slice := rb.ToSlice()
	if len(slice) != 3 {
		t.Fatalf("ToSlice() len = %d, want 3", len(slice))
	}
	if slice[0] != 1.1 || slice[1] != 2.2 || slice[2] != 3.3 {
		t.Errorf("ToSlice() = %v, want [1.1, 2.2, 3.3]", slice)
	}
}

func TestRingBuffer_SliceType(t *testing.T) {
	// Test with []float64 as element type (for per-core CPU history)
	rb := NewRingBuffer[[]float64](3)

	rb.Push([]float64{10, 20})
	rb.Push([]float64{30, 40})
	rb.Push([]float64{50, 60})

	slice := rb.ToSlice()
	if len(slice) != 3 {
		t.Fatalf("ToSlice() len = %d, want 3", len(slice))
	}
	if slice[0][0] != 10 || slice[0][1] != 20 {
		t.Errorf("slice[0] = %v, want [10, 20]", slice[0])
	}
	if slice[2][0] != 50 || slice[2][1] != 60 {
		t.Errorf("slice[2] = %v, want [50, 60]", slice[2])
	}
}

func BenchmarkRingBuffer_Push(b *testing.B) {
	rb := NewRingBuffer[float64](128)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rb.Push(float64(i))
	}
}

func BenchmarkSlice_AppendTrim(b *testing.B) {
	slice := make([]float64, 0, 128)
	historyLen := 128
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice = append(slice, float64(i))
		if len(slice) > historyLen {
			slice = slice[1:]
		}
	}
}
