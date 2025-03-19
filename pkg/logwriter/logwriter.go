package logwriter

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
)

// FlushIntervalMs is the interval in milliseconds at which the buffer is flushed to disk
const FlushIntervalMs = 1000

// LogWriter handles buffered writing of log lines to a file
type LogWriter struct {
	file       *os.File
	buffer     []*ds.LogLine
	bufferLock sync.Mutex
	stopChan   chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

// MakeLogWriter creates a new LogWriter that writes to the specified file
func MakeLogWriter(filename string) (*LogWriter, error) {
	// Open file with append mode, create if not exists
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	writer := &LogWriter{
		file:     file,
		buffer:   make([]*ds.LogLine, 0, 100),
		stopChan: make(chan struct{}),
	}

	// Start the background flushing goroutine
	writer.wg.Add(1)
	go writer.flushLoop()

	return writer, nil
}

// WriteLogLine adds a log line to the buffer
func (w *LogWriter) WriteLogLine(logLine *ds.LogLine) {
	w.bufferLock.Lock()
	defer w.bufferLock.Unlock()

	// Add to buffer
	w.buffer = append(w.buffer, logLine)
}

// flushLoop periodically flushes the buffer to disk
func (w *LogWriter) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(time.Duration(FlushIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.flush()
		case <-w.stopChan:
			// Final flush before exiting
			w.flush()
			return
		}
	}
}

// LogLineToString converts a LogLine to the persisted string format
// Format: linenum timestampmilli:logline
func LogLineToString(logLine ds.LogLine) string {
	nlMarker := ""
	if !strings.HasSuffix(logLine.Msg, "\n") {
		nlMarker = "\n"
	}
	return fmt.Sprintf("%d %d:%s%s", logLine.LineNum, logLine.Ts, logLine.Msg, nlMarker)
}

// StringToLogLine converts a persisted string back to a LogLine
// Format: linenum timestampmilli:logline
func StringToLogLine(line string) (ds.LogLine, error) {
	// Parse the line: linenum timestampmilli:logline
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ds.LogLine{}, fmt.Errorf("invalid log line format: %s", line)
	}

	// Parse the prefix (linenum timestampmilli)
	prefix := strings.Fields(parts[0])
	if len(prefix) != 2 {
		return ds.LogLine{}, fmt.Errorf("invalid log line prefix: %s", parts[0])
	}

	// Parse linenum
	lineNum, err := strconv.ParseInt(prefix[0], 10, 64)
	if err != nil {
		return ds.LogLine{}, fmt.Errorf("invalid line number: %s", prefix[0])
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(prefix[1], 10, 64)
	if err != nil {
		return ds.LogLine{}, fmt.Errorf("invalid timestamp: %s", prefix[1])
	}

	// Create LogLine
	return ds.LogLine{
		LineNum: lineNum,
		Ts:      ts,
		Msg:     parts[1],
	}, nil
}

// flush writes all buffered log lines to the file
func (w *LogWriter) flush() {
	w.bufferLock.Lock()
	defer w.bufferLock.Unlock()

	if len(w.buffer) == 0 {
		return
	}

	// Use a bytes.Buffer to accumulate all log lines
	var buf bytes.Buffer

	// Format each log line and add to buffer
	for _, logLine := range w.buffer {
		// Use the LogLineToString function to format the log line
		logEntry := LogLineToString(*logLine)
		buf.WriteString(logEntry)
	}

	// Write the entire buffer to file in a single operation
	if _, err := w.file.Write(buf.Bytes()); err != nil {
		// Just print the error and continue - we don't want to crash the app
		fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
	}

	// Clear the buffer
	w.buffer = w.buffer[:0]
}

// Dispose stops the writer and flushes any remaining logs
func (w *LogWriter) Dispose() error {
	// Signal the flush loop to stop (using sync.Once to ensure we only close the channel once)
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})

	// Wait for the flush loop to finish
	w.wg.Wait()

	// Close the file
	return w.file.Close()
}
