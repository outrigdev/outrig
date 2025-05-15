// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ValidatePollFunc validates that the provided function is suitable for use as a poll function.
// A valid poll function must:
// - Be non-nil
// - Be a function
// - Take 0 arguments
// - Return exactly 1 value
func ValidatePollFunc(fn any) error {
	if fn == nil {
		return fmt.Errorf("PollFunc requires a non-nil function")
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("PollFunc requires a function, got %s", fnType.Kind())
	}

	if fnType.NumIn() != 0 {
		return fmt.Errorf("PollFunc requires a function with 0 arguments, got %d", fnType.NumIn())
	}

	if fnType.NumOut() != 1 {
		return fmt.Errorf("PollFunc requires a function that returns exactly 1 value, got %d", fnType.NumOut())
	}

	return nil
}

// ValidatePollAtomic validates that the provided value is suitable for use as an atomic poll value.
// A valid atomic poll value must:
// - Be non-nil
// - Be a pointer
// - Point to a valid atomic type (sync/atomic package types or primitive types that support atomic operations)
func ValidatePollAtomic(val any) error {
	if val == nil {
		return fmt.Errorf("PollAtomic requires a non-nil value")
	}

	valType := reflect.TypeOf(val)
	// First, ensure we're dealing with a pointer
	if valType.Kind() != reflect.Ptr {
		return fmt.Errorf("PollAtomic requires a pointer to a value, got %s", valType.String())
	}

	// Get the element type (what the pointer points to)
	elemType := valType.Elem()
	typeName := elemType.String()

	// Check if val is a valid atomic type
	isValidAtomic := false

	// Check for atomic package types
	if strings.HasPrefix(typeName, "atomic.") {
		// Valid atomic types from sync/atomic package
		validAtomicTypes := map[string]bool{
			"atomic.Bool":    true,
			"atomic.Int32":   true,
			"atomic.Int64":   true,
			"atomic.Pointer": true,
			"atomic.Uint32":  true,
			"atomic.Uint64":  true,
			"atomic.Uintptr": true,
			"atomic.Value":   true,
		}
		isValidAtomic = validAtomicTypes[typeName]
	} else {
		// Check for primitive types that can be used with atomic operations
		switch elemType.Kind() {
		case reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			isValidAtomic = true
		}

		// Special case for unsafe.Pointer
		if typeName == "unsafe.Pointer" {
			isValidAtomic = true
		}
	}

	if !isValidAtomic {
		return fmt.Errorf("PollAtomic requires an atomic type, got %s", typeName)
	}

	return nil
}

// ValidatePollSync validates that the provided lock and value are suitable for use in a sync-based watch.
// A valid sync poll setup must have:
// - A non-nil sync.Locker
// - A non-nil value
// - The value must be a pointer
func ValidatePollSync(lock sync.Locker, val any) error {
	// Validate that lock is not nil
	if lock == nil {
		return fmt.Errorf("PollSync requires a non-nil sync.Locker")
	}

	// Validate that val is not nil
	if val == nil {
		return fmt.Errorf("PollSync requires a non-nil value")
	}

	// Validate that val is a pointer
	valType := reflect.TypeOf(val)
	if valType.Kind() != reflect.Ptr {
		return fmt.Errorf("PollSync requires a pointer to a value, got %s", valType.String())
	}

	return nil
}
