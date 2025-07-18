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
	lock       *sync.Mutex
	curVersion int64                   // Next version to assign
	m          map[K]versionedValue[V] // Map of values with versioning
}

func MakeVersionedMap[K comparable, V any]() *VersionedMap[K, V] {
	return &VersionedMap[K, V]{
		lock:       &sync.Mutex{},
		curVersion: 0,
		m:          make(map[K]versionedValue[V]),
	}
}

func (vm *VersionedMap[K, V]) SetVersion(version int64) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	if version > vm.curVersion {
		vm.curVersion = version
	}
}

func (vm *VersionedMap[K, V]) Set(key K, value V) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	vm.m[key] = versionedValue[V]{
		Version: vm.curVersion,
		Value:   value,
	}
}

func (vm *VersionedMap[K, V]) SetAndIncVersion(key K, value V) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	// Increment version for each new set operation
	vm.curVersion++
	vm.m[key] = versionedValue[V]{
		Version: vm.curVersion,
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
	return result, vm.curVersion
}

func (vm *VersionedMap[K, V]) ForEach(fn func(key K, value V, version int64)) {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	for key, versionedVal := range vm.m {
		fn(key, versionedVal.Value, versionedVal.Version)
	}
}

func (vm *VersionedMap[K, V]) Keys() []K {
	vm.lock.Lock()
	defer vm.lock.Unlock()

	keys := make([]K, 0, len(vm.m))
	for key := range vm.m {
		keys = append(keys, key)
	}
	return keys
}
