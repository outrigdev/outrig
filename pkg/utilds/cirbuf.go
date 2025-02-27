package utilds

import "sync"

// CirBuf is a generic circular buffer implementation that is thread-safe.
// It dynamically grows until it reaches MaxSize and can reclaim memory
// when emptied.
type CirBuf[T any] struct {
	Lock    *sync.Mutex
	MaxSize int
	Buf     []T
	Head    int
	Tail    int
}

// NewCirBuf creates a new circular buffer with the specified maximum size.
// The buffer is initially empty and will grow dynamically as elements are added.
func NewCirBuf[T any](maxSize int) *CirBuf[T] {
	return &CirBuf[T]{
		Lock:    &sync.Mutex{},
		MaxSize: maxSize,
		Buf:     nil,
		Head:    0,
		Tail:    0,
	}
}

// Write adds an element to the circular buffer.
// If the buffer is full, the oldest element will be overwritten.
// Returns a pointer to the element that was kicked out, or nil if no element was kicked out.
func (cb *CirBuf[T]) Write(element T) *T {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	if cb.Head == cb.Tail {
		// buffer is full (this also correctly handles the case when the buffer is nil, and size == 0)
		curSize := cb.size_nolock()
		if curSize == cb.MaxSize {
			// kick out the oldest element
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
