package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/outrigdev/goid"
)

func simpleWorker(id int) {
	fmt.Printf("[%s] Worker %2d started, goid: %2d\n", time.Now().Format("15:04:05"), id, goid.Get())
	time.Sleep(5 * time.Second)
}

func main() {
	// outrig.Init("goidtest", nil)
	// defer outrig.AppDone()

	fmt.Printf("Starting goid allocation test main-goid: %d\n", goid.Get())

	workerCount := 0
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()
	for {
		go simpleWorker(workerCount)
		workerCount++
		time.Sleep(2 * time.Second)
	}

}
