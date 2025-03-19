package boot

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// startViteServer starts the Vite development server as a subprocess
// and pipes its stdout/stderr to the Go server's stdout/stderr.
// It returns a function that can be called to stop the Vite server.
func startViteServer(ctx context.Context) (*exec.Cmd, error) {
	log.Printf("Starting Vite development server...\n")

	// Create the command to run task dev:vite
	cmd := exec.CommandContext(ctx, "task", "dev:vite")

	// Get pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Vite server: %w", err)
	}

	// Copy stdout to our stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf("[vite] %s\n", scanner.Text())
		}
	}()

	// Copy stderr to our stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, "[vite]", scanner.Text())
		}
	}()

	log.Printf("Vite development server started\n")
	return cmd, nil
}
