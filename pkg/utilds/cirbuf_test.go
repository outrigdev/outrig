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
