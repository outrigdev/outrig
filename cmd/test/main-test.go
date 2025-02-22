package main

import (
	_ "github.com/outrigdev/outrig/pkg/loginit"

	"fmt"
	"os"
	"time"

	"github.com/outrigdev/outrig"
)

func main() {
	fmt.Printf("log before init\n")
	outrig.Init(nil)
	defer outrig.Shutdown()
	fmt.Fprintf(os.Stderr, "[stderr] stdout is %T\n", os.Stdout)
	fmt.Printf("hello outrig!\n")
	time.Sleep(200 * time.Millisecond)
	outrig.Disable(false)
	fmt.Printf("during disable\n")
	time.Sleep(100 * time.Millisecond)
	outrig.Enable()
	fmt.Printf("after enable\n")
	time.Sleep(200 * time.Millisecond)
}
