// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"sync"
	"testing"
)

func TestCirBufBasicOperations(t *testing.T) {
	// Create a new circular buffer with max size 5
	cb := MakeCirBuf[int](5)

	// Check initial state
	if !cb.IsEmpty() {
		t.Error("New buffer should be empty")
	}
	if cb.IsFull() {
		t.Error("New buffer should not be full")
	}
	if cb.Size() != 0 {
		t.Errorf("Expected size 0, got %d", cb.Size())
	}

	// Write some elements
	cb.Write(10)
	cb.Write(20)
	cb.Write(30)

	// Check state after writing
	if cb.IsEmpty() {
		t.Error("Buffer should not be empty after writing")
	}
	if cb.IsFull() {
		t.Error("Buffer should not be full yet")
	}
	if cb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cb.Size())
	}

	// Read elements
	val, ok := cb.Read()
	if !ok || val != 10 {
		t.Errorf("Expected to read 10, got %d (ok: %v)", val, ok)
	}

	val, ok = cb.Read()
	if !ok || val != 20 {
		t.Errorf("Expected to read 20, got %d (ok: %v)", val, ok)
	}

	val, ok = cb.Read()
	if !ok || val != 30 {
		t.Errorf("Expected to read 30, got %d (ok: %v)", val, ok)
	}

	// Check if buffer is empty after reading all elements
	if !cb.IsEmpty() {
		t.Error("Buffer should be empty after reading all elements")
	}

	// Check if buffer is nil after reading all elements (memory reclamation)
	if cb.Buf != nil {
		t.Error("Buffer slice should be nil after reading all elements")
	}

	// Try to read from empty buffer
	_, ok = cb.Read()
	if ok {
		t.Errorf("Reading from empty buffer should return false, got %v", ok)
	}
}

func TestCirBufOverwrite(t *testing.T) {
	// Create a new circular buffer with max size 3
	cb := MakeCirBuf[string](3)

	// Write more elements than the max size
	kicked := cb.Write("A")
	if kicked != nil {
		t.Errorf("Expected nil kicked element, got %v", *kicked)
	}

	kicked = cb.Write("B")
	if kicked != nil {
		t.Errorf("Expected nil kicked element, got %v", *kicked)
	}

	kicked = cb.Write("C")
	if kicked != nil {
		t.Errorf("Expected nil kicked element, got %v", *kicked)
	}

	// Now the buffer is full, next writes should kick out elements
	kicked = cb.Write("D") // This should overwrite "A"
	if kicked == nil || *kicked != "A" {
		if kicked == nil {
			t.Error("Expected kicked element A, got nil")
		} else {
			t.Errorf("Expected kicked element A, got %v", *kicked)
		}
	}

	kicked = cb.Write("E") // This should overwrite "B"
	if kicked == nil || *kicked != "B" {
		if kicked == nil {
			t.Error("Expected kicked element B, got nil")
		} else {
			t.Errorf("Expected kicked element B, got %v", *kicked)
		}
	}

	// Check if buffer is full
	if !cb.IsFull() {
		t.Error("Buffer should be full")
	}

	// Read elements and verify the oldest elements were overwritten
	val, ok := cb.Read()
	if !ok || val != "C" {
		t.Errorf("Expected to read C, got %s (ok: %v)", val, ok)
	}

	val, ok = cb.Read()
	if !ok || val != "D" {
		t.Errorf("Expected to read D, got %s (ok: %v)", val, ok)
	}

	val, ok = cb.Read()
	if !ok || val != "E" {
		t.Errorf("Expected to read E, got %s (ok: %v)", val, ok)
	}

	// Buffer should be empty now
	if !cb.IsEmpty() {
		t.Error("Buffer should be empty after reading all elements")
	}
}

func TestCirBufConcurrency(t *testing.T) {
	// Create a new circular buffer with max size 100
	cb := MakeCirBuf[int](100)

	// Number of goroutines and operations
	numGoroutines := 10
	numOps := 100

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // For both readers and writers

	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				cb.Write(id*numOps + j)
			}
		}(i)
	}

	// Start reader goroutines
	readCount := 0
	var readMutex sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_, ok := cb.Read()
				if ok {
					readMutex.Lock()
					readCount++
					readMutex.Unlock()
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Verify that we've read some elements
	// Note: We can't predict exactly how many due to the concurrent nature
	t.Logf("Read %d elements out of %d written", readCount, numGoroutines*numOps)

	// Read any remaining elements
	remaining := 0
	for {
		_, ok := cb.Read()
		if !ok {
			break
		}
		remaining++
	}

	t.Logf("Read %d remaining elements", remaining)
	t.Logf("Total elements read: %d", readCount+remaining)

	// Buffer should be empty now
	if !cb.IsEmpty() {
		t.Error("Buffer should be empty after reading all elements")
	}
}

func TestCirBufWriteAt(t *testing.T) {
	// Create a new circular buffer with max size 5
	cb := MakeCirBuf[int](5)

	// Write some initial elements
	cb.Write(10)
	cb.Write(20)
	cb.Write(30)
	// Buffer now has indexes 0, 1, 2 with values 10, 20, 30

	// Test writing within existing range
	err := cb.WriteAt(25, 1)
	if err != nil {
		t.Errorf("WriteAt within range should not error: %v", err)
	}

	// Verify the overwrite worked
	all, headOffset := cb.GetAll()
	expected := []int{10, 25, 30}
	if len(all) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(all))
	}
	for i, val := range expected {
		if all[i] != val {
			t.Errorf("Expected all[%d] = %d, got %d", i, val, all[i])
		}
	}
	if headOffset != 0 {
		t.Errorf("Expected headOffset 0, got %d", headOffset)
	}

	// Test writing before buffer range (should error)
	err = cb.WriteAt(99, -1)
	if err == nil {
		t.Error("WriteAt before buffer range should return error")
	}

	// Test writing beyond current range (should fill with zeros)
	err = cb.WriteAt(50, 6)
	if err != nil {
		t.Errorf("WriteAt beyond range should not error: %v", err)
	}

	// Verify zeros were filled and element was written
	// Since max size is 5 and we're writing 7 elements total, oldest 2 should be kicked out
	all, headOffset = cb.GetAll()
	expected = []int{30, 0, 0, 0, 50}
	if len(all) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(all))
	}
	for i, val := range expected {
		if all[i] != val {
			t.Errorf("Expected all[%d] = %d, got %d", i, val, all[i])
		}
	}
	if headOffset != 2 {
		t.Errorf("Expected headOffset 2, got %d", headOffset)
	}

	// Test writing beyond max size (should kick out oldest elements)
	cb2 := MakeCirBuf[int](3)
	cb2.Write(1)
	cb2.Write(2)
	cb2.Write(3)
	// Buffer is full with indexes 0, 1, 2

	err = cb2.WriteAt(99, 5)
	if err != nil {
		t.Errorf("WriteAt beyond max size should not error: %v", err)
	}

	// Should have kicked out oldest elements and filled with zeros
	all, _ = cb2.GetAll()
	expected = []int{0, 0, 99}
	if len(all) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(all))
	}
	for i, val := range expected {
		if all[i] != val {
			t.Errorf("Expected all[%d] = %d, got %d", i, val, all[i])
		}
	}
	last, _, _ := cb2.GetLast()
	if last != 99 {
		t.Errorf("Expected last element 99, got %d", last)
	}
	first, _, _ := cb2.GetFirst()
	if first != 0 {
		t.Errorf("Expected first element 0, got %d", first)
	}

	cb2.Write(4)
	expected = []int{0, 99, 4}
	all, _ = cb2.GetAll()
	if len(all) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(all))
	}
	for i, val := range expected {
		if all[i] != val {
			t.Errorf("Expected all[%d] = %d, got %d", i, val, all[i])
		}
	}
}
