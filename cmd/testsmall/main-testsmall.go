package main

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/outrigdev/outrig"
)

func main() {
	// Initialize Outrig with default configuration
	config := outrig.DefaultDevConfig()
	outrig.Init(config)
	defer outrig.AppDone()

	// Print a hello world log line
	fmt.Printf("Hello, world!\n")

	// Print a blank log line
	fmt.Printf("\n\n")

	// print a stack trace
	debug.PrintStack()
	time.Sleep(1 * time.Millisecond)

	for i := 1; i <= 10; i++ {
		fmt.Printf("Line %d\n", i)
		time.Sleep(500 * time.Millisecond)
	}
}
