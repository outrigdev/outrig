// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/outrigdev/outrig/autoinit" // Auto-initialize Outrig
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Printf("Starting simple demo application\n")
	fmt.Printf("This is a regular Go program that just added one import line!\n")

	log.Printf("Application started")
	log.Printf("Processing some work...")

	// Simulate some typical application work
	for i := 1; i <= 10; i++ {
		processItem(i)
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("All work completed")
	fmt.Printf("Demo finished! Check the Outrig UI to see captured logs.\n")
}

func processItem(id int) {
	log.Printf("Processing item %d", id)
	
	// Simulate some work with random delays
	workTime := time.Duration(rand.Intn(200)+50) * time.Millisecond
	time.Sleep(workTime)
	
	if rand.Float32() < 0.1 {
		log.Printf("Warning: Item %d took longer than expected (%v)", id, workTime)
	}
	
	log.Printf("Completed item %d in %v", id, workTime)
}