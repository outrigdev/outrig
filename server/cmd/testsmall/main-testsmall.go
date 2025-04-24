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

func main() {
	config := configpkg.DefaultConfigForOutrigDevelopment()
	config.LogProcessorConfig.OutrigPath = "go"
	config.LogProcessorConfig.AdditionalArgs = []string{"run", "server/main-server.go"}
	outrig.Init("test-small", config)
	defer outrig.AppDone()

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
