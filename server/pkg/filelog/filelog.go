// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// for internal testing of log capture
package filelog

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const LogFilePath = "/tmp/outrig.log"

var fileLogger *os.File
var initOnce sync.Once

// Logf writes a message to the file logger
func Logf(format string, args ...interface{}) {
	initOnce.Do(func() {
		var err error
		fileLogger, err = os.OpenFile(LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", LogFilePath, err)
		}
	})

	output := fileLogger
	if output == nil {
		output = os.Stderr
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	if len(message) == 0 || message[len(message)-1] != '\n' {
		message += "\n"
	}
	fmt.Fprintf(output, "[%s] %s", timestamp, message)
}
