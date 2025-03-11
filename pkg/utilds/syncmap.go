// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import "sync"

type SyncMap[T any] struct {
	lock *sync.Mutex
	m    map[string]T
}

func MakeSyncMap[T any]() *SyncMap[T] {
	return &SyncMap[T]{
		lock: &sync.Mutex{},
		m:    make(map[string]T),
	}
}

func (sm *SyncMap[T]) Set(key string, value T) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.m[key] = value
}

func (sm *SyncMap[T]) Get(key string) T {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	return sm.m[key]
}

func (sm *SyncMap[T]) GetEx(key string) (T, bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	v, ok := sm.m[key]
	return v, ok
}

func (sm *SyncMap[T]) Delete(key string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.m, key)
}

// GetOrCreate gets a value by key. If the key doesn't exist, it calls the provided
// function to create a new value, sets it in the map, and returns it.
// Returns the value and a boolean indicating if the key was found (true) or created (false).
func (sm *SyncMap[T]) GetOrCreate(key string, createFn func() T) (T, bool) {
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
func (sm *SyncMap[T]) Keys() []string {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	
	keys := make([]string, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}
	
	return keys
}

// Len returns the number of items in the map
func (sm *SyncMap[T]) Len() int {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	
	return len(sm.m)
}
