// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/outrigdev/outrig"
	configpkg "github.com/outrigdev/outrig/pkg/config"
	"github.com/sirupsen/logrus"
)

// Global Watch variables initialized at package level
var nilWatch = outrig.NewWatch("test-nil").WithTags("test").AsJSON().ForPush()
var fooWatch = outrig.NewWatch("foo").WithTags("test").ForPush()
var mapWatch = outrig.NewWatch("map").WithTags("test").ForPush()
var strWatch = outrig.NewWatch("str").WithTags("test").AsJSON().ForPush()

type Point struct {
	X    int
	Y    int
	Desc string
}

func (p Point) String() string {
	return fmt.Sprintf("[%d,%d]", p.X, p.Y)
}

type Foo struct {
	Val       int
	Ch        chan int
	SubStruct Point
}

func (f Foo) String() string {
	return fmt.Sprintf("Foo=%d, Ch=%v, SubStruct=%s", f.Val, f.Ch, f.SubStruct)
}

func main() {
	config := configpkg.DefaultConfigForOutrigDevelopment()
	config.LogProcessorConfig.OutrigPath = "go"
	config.LogProcessorConfig.AdditionalArgs = []string{"run", "server/main-server.go"}
	outrig.Init("test-small", config)
	defer outrig.AppDone()

	// Create temporary log file
	tmpFile, err := os.CreateTemp("", "outrig-testsmall-*.log")
	if err != nil {
		fmt.Printf("Failed to create temp log file: %v\n", err)
		return
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Set up logger to write to both temp file and outrig stream
	multiWriter := io.MultiWriter(tmpFile, outrig.MakeLogStream("tmplog"))
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Log some test messages
	log.Printf("Starting testsmall application")
	log.Printf("Temp log file created at: %s", tmpFile.Name())
	log.Printf("Current working directory: %s", filepath.Dir("."))

	fmt.Printf("Logging to temp file: %s\n", tmpFile.Name())

	// Push nil value using global watch
	nilWatch.Push(nil)

	// Create and push foo value using global watch
	foo := &Foo{5, make(chan int, 2), Point{1, 2, "test{[()]}"}}
	fooWatch.Push(foo)

	strWatch.Push("test string :pizza:")

	// Create and push map value using global watch
	m := make(map[string]any)
	m["test"] = 1
	m["foo"] = 55
	m["struct"] = Point{1, 2, "point-struct"}
	m["arr"] = []int{1, 2, 3}
	mapWatch.Push(m)

	outrig.Logf("#test this is a log line from outrig :warning: logger :apple: :pizza:")
	outrig.Logf("#test another log line")
	for i := 0; i < 5; i++ {
		outrig.Logf("#test log line %d", i)
	}
	outrig.Logf("#test long log line that has more text than the default length of 80 characters. This is a test to see how the log line is shown and displayed in the output if it exceeds the maximum length.")

	ow := outrig.MakeLogStream("hellohello")
	bow := bufio.NewWriter(ow)
	bow.WriteString("Hello, world!\n")
	fmt.Fprintf(bow, "This is a \033[30;43mtest log\033[0m line with a \033[3mnumber\033[0m: %d\n", 42)
	bow.Flush()

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	// Print a hello world log line
	logrus.Info("Hello, world!")

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
