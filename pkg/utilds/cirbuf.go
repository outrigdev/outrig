package utilds

import (
	"sync"
)

// CirBuf is a generic circular buffer implementation that is thread-safe.
// It dynamically grows until it reaches MaxSize and can reclaim memory
// when emptied.
type CirBuf[T any] struct {
	Lock       *sync.Mutex
	MaxSize    int
	TotalCount int
	HeadOffset int
	Buf        []T
	Head       int
	Tail       int
}

// MakeCirBuf creates a new circular buffer with the specified maximum size.
// The buffer is initially empty and will grow dynamically as elements are added.
func MakeCirBuf[T any](maxSize int) *CirBuf[T] {
	return &CirBuf[T]{
		Lock:       &sync.Mutex{},
		MaxSize:    maxSize,
		TotalCount: 0,
		HeadOffset: 0,
		Buf:        nil,
		Head:       0,
		Tail:       0,
	}
}

// Write adds an element to the circular buffer.
// If the buffer is full, the oldest element will be overwritten.
// Returns a pointer to the element that was kicked out, or nil if no element was kicked out.
func (cb *CirBuf[T]) Write(element T) *T {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	cb.TotalCount++
	if cb.Head == cb.Tail {
		// buffer is full (this also correctly handles the case when the buffer is nil, and size == 0)
		curSize := cb.size_nolock()
		if curSize == cb.MaxSize {
			// kick out the oldest element
			cb.HeadOffset++
			kickedOut := cb.Buf[cb.Head]
			cb.Buf[cb.Head] = element
			cb.Head = (cb.Head + 1) % len(cb.Buf)
			cb.Tail = cb.Head
			return &kickedOut
		}
		newBuf := make([]T, max(min(curSize*2, cb.MaxSize), 1))
		copy(newBuf, cb.Buf[cb.Head:])
		copy(newBuf[len(cb.Buf)-cb.Head:], cb.Buf[:cb.Head])
		cb.Buf = newBuf
		cb.Head = 0
		cb.Tail = curSize
		// fall through to actually write the element
	}
	// otherwise buffer is not full, write the next element
	cb.Buf[cb.Tail] = element
	cb.Tail = (cb.Tail + 1) % len(cb.Buf)
	return nil
}

// Read removes and returns the oldest element from the circular buffer.
// If the buffer is empty, the zero value of T and false are returned.
func (cb *CirBuf[T]) Read() (T, bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock() // get the current number of elements
	if size == 0 {
		var zero T
		return zero, false
	}

	cb.HeadOffset++
	elem := cb.Buf[cb.Head]
	if size == 1 {
		cb.Buf = nil
		cb.Head = 0
		cb.Tail = 0
	} else {
		cb.Head = (cb.Head + 1) % len(cb.Buf)
	}
	return elem, true
}

// size_nolock returns the current number of elements in the buffer without acquiring the lock.
// This is an internal helper function and should only be called when the lock is already held.
func (cb *CirBuf[T]) size_nolock() int {
	if cb.Buf == nil {
		return 0
	}
	if cb.Head == cb.Tail {
		return len(cb.Buf)
	}
	if cb.Head < cb.Tail {
		return cb.Tail - cb.Head
	}
	return len(cb.Buf) - cb.Head + cb.Tail
}

// Size returns the current number of elements in the buffer.
func (cb *CirBuf[T]) Size() int {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()
	return cb.size_nolock()
}

// IsEmpty returns true if the buffer is empty.
func (cb *CirBuf[T]) IsEmpty() bool {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()
	return cb.size_nolock() == 0
}

// IsFull returns true if the buffer is full.
func (cb *CirBuf[T]) IsFull() bool {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()
	return cb.size_nolock() == cb.MaxSize
}

// GetAll returns a slice containing all elements in the buffer in order from oldest to newest.
// This does not remove elements from the buffer.
func (cb *CirBuf[T]) GetAll() []T {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		return []T{}
	}

	result := make([]T, size)

	if cb.Buf == nil {
		return result
	}

	// Copy elements from head to end of buffer
	if cb.Head < cb.Tail {
		copy(result, cb.Buf[cb.Head:cb.Tail])
	} else {
		// Copy elements from head to end of underlying array
		n := copy(result, cb.Buf[cb.Head:])
		// Copy elements from start of underlying array to tail
		copy(result[n:], cb.Buf[:cb.Tail])
	}

	return result
}

func (cb *CirBuf[T]) GetTotalCountAndHeadOffset() (int, int) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	return cb.TotalCount, cb.HeadOffset
}

// returns items, true-offset, eof
func (cb *CirBuf[T]) GetRange(start int, end int) ([]T, int, bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()
	if start < cb.HeadOffset {
		start = cb.HeadOffset
	}
	var eof bool
	if end >= cb.TotalCount {
		end = cb.TotalCount
		eof = true
	}
	realStartOffset := start - cb.HeadOffset
	realEndOffset := end - cb.HeadOffset
	rtnCount := realEndOffset - realStartOffset
	if rtnCount == 0 {
		return nil, start, eof
	}
	startPos := (cb.Head + realStartOffset) % len(cb.Buf)
	rtn := make([]T, rtnCount)
	for i := 0; i < rtnCount; i++ {
		offset := (startPos + i) % len(cb.Buf)
		rtn[i] = cb.Buf[offset]
	}
	return rtn, start, eof
}

// GetLast returns the last element in the buffer, its offset, and a boolean indicating
// whether the buffer has any elements. If the buffer is empty, the zero value of T,
// 0, and false are returned.
func (cb *CirBuf[T]) GetLast() (T, int, bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		var zero T
		return zero, 0, false
	}

	// Calculate the position of the last element
	lastPos := (cb.Tail - 1 + len(cb.Buf)) % len(cb.Buf)
	
	// The offset is simply TotalCount - 1 (index of the last element)
	lastOffset := cb.TotalCount - 1
	
	return cb.Buf[lastPos], lastOffset, true
}
