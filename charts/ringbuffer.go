package charts

import "math"

// RingBuffer is a fixed-capacity circular buffer of float64 samples designed
// for zero-allocation real-time telemetry feeding AreaSparkline.
//
// Samples are stored in chronological order from oldest (index 0) to newest
// (index Len()-1). Pushing when full overwrites the oldest entry in O(1) time
// with zero heap allocations. NaN values are preserved and act as curve gaps.
type RingBuffer struct {
	data  []float64
	head  int
	count int
	seq   uint64
}

// NewRingBuffer creates a RingBuffer with the given sample capacity. A
// non-positive capacity creates a safe empty buffer whose Push is a no-op.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		return &RingBuffer{}
	}
	return &RingBuffer{
		data: make([]float64, capacity),
	}
}

// Cap returns the maximum number of samples the buffer holds.
func (r *RingBuffer) Cap() int {
	if r == nil {
		return 0
	}
	return len(r.data)
}

// Len returns the number of samples currently stored (0..Cap()).
func (r *RingBuffer) Len() int {
	if r == nil {
		return 0
	}
	return r.count
}

// Sequence returns a monotonically increasing mutation counter incremented on
// every Push, Clear, or Reset.
func (r *RingBuffer) Sequence() uint64 {
	if r == nil {
		return 0
	}
	return r.seq
}

// Push appends v to the buffer in O(1) time with zero allocations, evicting
// the oldest sample when at capacity.
func (r *RingBuffer) Push(v float64) {
	if r == nil || len(r.data) == 0 {
		return
	}
	if r.count < len(r.data) {
		idx := (r.head + r.count) % len(r.data)
		r.data[idx] = v
		r.count++
	} else {
		r.data[r.head] = v
		r.head++
		if r.head == len(r.data) {
			r.head = 0
		}
	}
	r.seq++
}

// PushSlice appends every element of vs in order without allocating.
func (r *RingBuffer) PushSlice(vs ...float64) {
	if r == nil || len(r.data) == 0 {
		return
	}
	for _, v := range vs {
		r.Push(v)
	}
}

// Clear empties the buffer while retaining its backing storage.
func (r *RingBuffer) Clear() {
	if r == nil {
		return
	}
	r.head = 0
	r.count = 0
	r.seq++
}

// Reset clears the buffer and ensures it has the requested capacity.
func (r *RingBuffer) Reset(capacity int) {
	if r == nil {
		return
	}
	if capacity <= 0 {
		r.data = nil
	} else if cap(r.data) >= capacity {
		r.data = r.data[:capacity]
	} else {
		r.data = make([]float64, capacity)
	}
	r.head = 0
	r.count = 0
	r.seq++
}

// At returns the i-th sample from oldest (0) to newest (Len()-1), or NaN when
// r is nil or i is out of range.
func (r *RingBuffer) At(i int) float64 {
	if r == nil || i < 0 || i >= r.count {
		return math.NaN()
	}
	idx := r.head + i
	if idx >= len(r.data) {
		idx -= len(r.data)
	}
	return r.data[idx]
}

// Latest returns the most recently pushed sample, or (0, false) if empty.
func (r *RingBuffer) Latest() (float64, bool) {
	if r == nil || r.count == 0 {
		return 0, false
	}
	return r.At(r.count - 1), true
}

// MinMax returns the minimum and maximum finite values stored in the buffer,
// skipping NaN and infinite samples. ok is false if no finite sample exists.
func (r *RingBuffer) MinMax() (lo, hi float64, ok bool) {
	if r == nil || r.count == 0 {
		return 0, 0, false
	}
	lo, hi = math.Inf(1), math.Inf(-1)
	for i := range r.count {
		v := r.At(i)
		if !isFiniteF64(v) {
			continue
		}
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
		ok = true
	}
	if !ok {
		return 0, 0, false
	}
	return lo, hi, true
}

// CopyTo writes all stored samples in chronological order into dst[:0] and
// returns the populated slice. When cap(dst) >= r.Len(), CopyTo performs zero
// allocations.
func (r *RingBuffer) CopyTo(dst []float64) []float64 {
	dst = dst[:0]
	if r == nil || r.count == 0 {
		return dst
	}
	first := min(r.count, len(r.data)-r.head)
	dst = append(dst, r.data[r.head:r.head+first]...)
	if rem := r.count - first; rem > 0 {
		dst = append(dst, r.data[:rem]...)
	}
	return dst
}
