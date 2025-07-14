// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"fmt"
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// write_nolock adds an element to the circular buffer without acquiring the lock.
// This is an internal helper function and should only be called when the lock is already held.
// Returns a pointer to the element that was kicked out, or nil if no element was kicked out.
func (cb *CirBuf[T]) write_nolock(element T) *T {
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

// Write adds an element to the circular buffer.
// If the buffer is full, the oldest element will be overwritten.
// Returns a pointer to the element that was kicked out, or nil if no element was kicked out.
func (cb *CirBuf[T]) Write(element T) *T {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()
	return cb.write_nolock(element)
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
// It also returns the HeadOffset, which is the offset of the first element in the buffer.
func (cb *CirBuf[T]) GetAll() ([]T, int) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		return []T{}, cb.HeadOffset
	}

	result := make([]T, size)

	if cb.Buf == nil {
		return result, cb.HeadOffset
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

	return result, cb.HeadOffset
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
	if rtnCount <= 0 {
		return nil, start, eof
	}
	if len(cb.Buf) == 0 {
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

// GetFirst returns the first element in the buffer, its offset, and a boolean indicating
// whether the buffer has any elements. If the buffer is empty, the zero value of T,
// 0, and false are returned.
func (cb *CirBuf[T]) GetFirst() (T, int, bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		var zero T
		return zero, 0, false
	}

	return cb.Buf[cb.Head], cb.HeadOffset, true
}

// FilterItems returns a slice of items for which the provided filter function returns true.
// The filter function takes the item and its absolute index (TotalCount-based) and returns a boolean.
// This is useful for filtering items based on custom criteria, such as timestamp.
func (cb *CirBuf[T]) FilterItems(filter func(item T, index int) bool) []T {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		return []T{}
	}

	var result []T

	// Calculate the absolute index of the first item in the buffer
	firstIndex := cb.TotalCount - size

	// Iterate through all items in the buffer
	for i := 0; i < size; i++ {
		// Calculate the position in the underlying array
		pos := (cb.Head + i) % len(cb.Buf)

		// Calculate the absolute index of this item
		absIndex := firstIndex + i

		// Apply the filter function
		if filter(cb.Buf[pos], absIndex) {
			result = append(result, cb.Buf[pos])
		}
	}

	return result
}

// ForEach iterates through all elements in the buffer in order from oldest to newest,
// calling the provided function on each element. If the function returns false,
// iteration stops early. The buffer is locked during the entire iteration.
func (cb *CirBuf[T]) ForEach(fn func(item T) bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	size := cb.size_nolock()
	if size == 0 {
		return
	}

	if cb.Buf == nil {
		return
	}

	// Iterate through elements from head to tail (oldest to newest)
	for i := 0; i < size; i++ {
		pos := (cb.Head + i) % len(cb.Buf)
		if !fn(cb.Buf[pos]) {
			break
		}
	}
}

// WriteAt writes an element at a specific index in the buffer.
// Returns an error if the index is before the current buffer range.
// If the index is beyond the current range, fills with zero values up to that index.
// If the index is within the current range, overwrites the element at that position.
func (cb *CirBuf[T]) WriteAt(element T, index int) error {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	// Error if trying to write before the current buffer range
	if index < cb.HeadOffset {
		return fmt.Errorf("cannot write at index %d: before buffer range (starts at %d)", index, cb.HeadOffset)
	}

	// If writing within the current range, just overwrite
	if index < cb.TotalCount {
		bufferIndex := index - cb.HeadOffset
		pos := (cb.Head + bufferIndex) % len(cb.Buf)
		cb.Buf[pos] = element
		return nil
	}

	// If writing beyond current range, fill with zeros up to the target index
	var zero T
	gapSize := index - cb.TotalCount

	// Write zeros to fill the gap
	for i := 0; i < gapSize; i++ {
		cb.write_nolock(zero)
	}

	// Write the actual element
	cb.write_nolock(element)

	return nil
}

// GetAt returns the element at the specified absolute index and a boolean indicating
// whether the element exists. The index is absolute (based on TotalCount), not relative
// to the current buffer position. Returns the zero value of T and false if the index
// is out of bounds.
func (cb *CirBuf[T]) GetAt(index int) (T, bool) {
	cb.Lock.Lock()
	defer cb.Lock.Unlock()

	// Check if index is within valid range
	if index < cb.HeadOffset || index >= cb.TotalCount {
		var zero T
		return zero, false
	}

	// Convert absolute index to buffer position
	bufferIndex := index - cb.HeadOffset
	pos := (cb.Head + bufferIndex) % len(cb.Buf)

	return cb.Buf[pos], true
}
