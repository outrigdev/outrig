// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"sync"
)

type versionedValue[V any] struct {
	Version int64 // Version of the value
	Value   V     // The actual value
}

type VersionedMap[K comparable, V any] struct {
	lock        *sync.Mutex
	nextVersion int64                        // Next version to assign
	m           map[K]versionedValue[V]      // Map of values with versioning
}

func MakeVersionedMap[K comparable, V any]() *VersionedMap[K, V] {
	return &VersionedMap[K, V]{
		lock:        &sync.Mutex{},
		nextVersion: 0,
		m:           make(map[K]versionedValue[V]),
	}
}

func (vm *VersionedMap[K, V]) Set(key K, value V) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	// Increment version for each new set operation
	vm.nextVersion++
	vm.m[key] = versionedValue[V]{
		Version: vm.nextVersion,
		Value:   value,
	}
}

func (vm *VersionedMap[K, V]) Get(key K) (V, int64, bool) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	versionedVal, exists := vm.m[key]
	if !exists {
		var zero V
		return zero, 0, false
	}
	return versionedVal.Value, versionedVal.Version, true
}

func (vm *VersionedMap[K, V]) GetSinceVersion(version int64) (map[K]V, int64) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	result := make(map[K]V)
	for key, versionedVal := range vm.m {
		if versionedVal.Version > version {
			result[key] = versionedVal.Value
		}
	}
	return result, vm.nextVersion
}