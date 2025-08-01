// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import "sync"

type SyncMap[K comparable, T any] struct {
	lock *sync.Mutex
	m    map[K]T
}

func MakeSyncMap[K comparable, T any]() *SyncMap[K, T] {
	return &SyncMap[K, T]{
		lock: &sync.Mutex{},
		m:    make(map[K]T),
	}
}

func (sm *SyncMap[K, T]) Set(key K, value T) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.m[key] = value
}

func (sm *SyncMap[K, T]) Get(key K) T {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	return sm.m[key]
}

func (sm *SyncMap[K, T]) GetEx(key K) (T, bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	v, ok := sm.m[key]
	return v, ok
}

func (sm *SyncMap[K, T]) Delete(key K) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.m, key)
}

// GetOrCreate gets a value by key. If the key doesn't exist, it calls the provided
// function to create a new value, sets it in the map, and returns it.
// Returns the value and a boolean indicating if the key was found (true) or created (false).
func (sm *SyncMap[K, T]) GetOrCreate(key K, createFn func() T) (T, bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if val, ok := sm.m[key]; ok {
		return val, true
	}

	// Key doesn't exist, create new value
	newVal := createFn()
	sm.m[key] = newVal
	return newVal, false
}

// Keys returns a slice of all keys in the map
func (sm *SyncMap[K, T]) Keys() []K {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	keys := make([]K, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}

	return keys
}

// Len returns the number of items in the map
func (sm *SyncMap[K, T]) Len() int {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	return len(sm.m)
}

// ForEach iterates over all key/value pairs in the map, calling the provided function for each pair
func (sm *SyncMap[K, T]) ForEach(fn func(K, T)) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for k, v := range sm.m {
		fn(k, v)
	}
}
