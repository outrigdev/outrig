package main

import (
	"fmt"
	"time"
)

func worker(id int, duration time.Duration) {
	fmt.Printf("Worker %d is working for %v\n", id, duration)
	time.Sleep(duration)
	fmt.Printf("Worker %d finished\n", id)
}

func longRunningTask(name string, duration time.Duration) {
	fmt.Printf("Long running task '%s' started for %v\n", name, duration)
	time.Sleep(duration)
	fmt.Printf("Long running task '%s' finished\n", name)
}

func periodicTask(name string, interval time.Duration, totalDuration time.Duration) {
	fmt.Printf("Periodic task '%s' started (interval: %v, total: %v)\n", name, interval, totalDuration)
	start := time.Now()
	for time.Since(start) < totalDuration {
		fmt.Printf("Periodic task '%s' tick\n", name)
		time.Sleep(interval)
	}
	fmt.Printf("Periodic task '%s' finished\n", name)
}

func main() {
	fmt.Println("Starting dynamic 30-second test...")
	start := time.Now()

	// Initial batch of workers with different durations
	for i := 0; i < 3; i++ {
		duration := time.Duration(2+i*2) * time.Second
		//outrig name="initial-worker"
		go worker(i, duration)
	}

	// Long running background task
	//outrig name="background-processor"
	go longRunningTask("background-processor", 25*time.Second)

	// Periodic task that runs throughout most of the test
	//outrig name="periodic-heartbeat"
	go periodicTask("heartbeat", 3*time.Second, 28*time.Second)

	// Staggered short tasks
	go func() {
		for i := 0; i < 8; i++ {
			time.Sleep(time.Duration(2+i) * time.Second)
			taskId := i + 10
			duration := time.Duration(1+i%3) * time.Second
			//outrig name="staggered-task"
			go worker(taskId, duration)
		}
	}()

	// Mid-test burst of activity
	go func() {
		time.Sleep(12 * time.Second)
		fmt.Println("Mid-test burst starting...")
		for i := 0; i < 5; i++ {
			taskId := i + 100
			duration := time.Duration(1+i%4) * time.Second
			//outrig name="burst-worker"
			go worker(taskId, duration)
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Anonymous tasks with varying patterns
	//outrig name="quick-task"
	go func() {
		for i := 0; i < 6; i++ {
			fmt.Printf("Quick task iteration %d\n", i)
			time.Sleep(1 * time.Second)
		}
	}()

	//outrig name="variable-sleep-task"
	go func(multiplier int) {
		for i := 0; i < 4; i++ {
			sleepTime := time.Duration(multiplier*(i+1)) * time.Second
			fmt.Printf("Variable sleep task sleeping for %v\n", sleepTime)
			time.Sleep(sleepTime)
		}
	}(2)

	// Late starting tasks
	go func() {
		time.Sleep(20 * time.Second)
		fmt.Println("Late tasks starting...")
		//outrig name="late-worker"
		go worker(200, 8*time.Second)
		
		//outrig name="final-sprint"
		go func() {
			for i := 0; i < 3; i++ {
				fmt.Printf("Final sprint iteration %d\n", i)
				time.Sleep(2 * time.Second)
			}
		}()
	}()

	// This go statement has no outrig directive, so it won't be transformed
	func() {
		go func() {
			fmt.Println("Regular goroutine (not monitored)")
			time.Sleep(5 * time.Second)
		}()
	}()

	fmt.Println("All initial workers started, test will run for 30 seconds...")
	
	// Progress indicator
	go func() {
		for elapsed := 5; elapsed <= 30; elapsed += 5 {
			time.Sleep(5 * time.Second)
			fmt.Printf("Test progress: %d/30 seconds elapsed\n", elapsed)
		}
	}()

	time.Sleep(30 * time.Second)
	fmt.Printf("Test completed after %v\n", time.Since(start))
}
