// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/outrigdev/outrig"
	configpkg "github.com/outrigdev/outrig/pkg/config"
	"github.com/sirupsen/logrus"
)

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

	outrig.TrackValue("test #test", nil)

	foo := &Foo{5, make(chan int, 2), Point{1, 2, "test{[()]}"}}
	outrig.TrackValue("foo #test", foo)

	m := make(map[string]any)
	m["test"] = 1
	m["foo"] = 55
	m["struct"] = Point{1, 2, "point-struct"}
	m["arr"] = []int{1, 2, 3}

	outrig.TrackValue("map #test", m)

	outrig.Logf("#test: this is a log line from outrig :warning: logger :apple: :pizza:")

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
