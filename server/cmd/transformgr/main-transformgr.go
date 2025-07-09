package main

import (
	"fmt"
	"time"
)

func worker(id int) {
	fmt.Printf("Worker %d is working\n", id)
	time.Sleep(3 * time.Second) // Simulate work
}

func main() {
	fmt.Println("Starting workers...")

	//outrig name="worker-1"
	go worker(1)

	//outrig name="worker-2"
	go worker(2)

	//outrig name="anonymous-task"
	go func() {
		fmt.Println("Anonymous task running")
		time.Sleep(1 * time.Second) // Simulate work
	}()

	//outrig name="parameterized-task"
	go func(x int) {
		fmt.Printf("Parameterized task with value: %d\n", x)
		time.Sleep(2 * time.Second) // Simulate work
	}(55)

	// This go statement has no outrig directive, so it won't be transformed
	func() {
		go func() {
			fmt.Println("Regular goroutine")
			time.Sleep(2 * time.Second) // Simulate work
		}()
	}()

	fmt.Println("All workers started")
	time.Sleep(4 * time.Second) // Wait for all workers to finish
}
