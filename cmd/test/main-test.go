package main

import (
	"fmt"
	"time"

	"github.com/outrigdev/outrig"
)

func main() {
	fmt.Printf("log before init\n")
	config := outrig.DefaultConfig()
	config.LogProcessorConfig.WrapStderr = false
	outrig.Init(config)
	defer outrig.AppDone()
	fmt.Printf("hello outrig!\n")
	time.Sleep(200 * time.Millisecond)
	outrig.Disable(false)
	fmt.Printf("during disable\n")
	time.Sleep(100 * time.Millisecond)
	outrig.Enable()
	fmt.Printf("after enable\n")
	fmt.Printf("again\n")

	// Loop that outputs a new log line every second until program is terminated
	counter := 0
	for {
		fmt.Printf("Counter: %d\n", counter)
		counter++
		time.Sleep(5 * time.Millisecond)
	}
}
