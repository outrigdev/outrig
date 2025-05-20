package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

var (
	// Server process
	serverCmd  *exec.Cmd
	serverLock sync.Mutex
)

//go:embed assets/outrigapp-padded.png
var iconData []byte

// getOutrigPath returns the path to the outrig executable
func getOutrigPath() string {
	// Always use the outrig in the same directory
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
		return "outrig"
	}

	return filepath.Join(filepath.Dir(execPath), "outrig")
}

// startServer starts the Outrig server
func startServer() {
	serverLock.Lock()
	defer serverLock.Unlock()

	log.Println("Starting Outrig server...")

	// Get the path to the outrig executable
	outrigPath := getOutrigPath()

	// Start the server with close-on-stdin flag
	serverCmd = exec.Command(outrigPath, "server", "--close-on-stdin")

	// Create a pipe for stdin
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		log.Printf("Error creating stdin pipe: %v", err)
		return
	}

	// We keep stdin open, but if outrigapp crashes, it will close automatically
	// causing the server to shut down due to the --close-on-stdin flag

	// Set up stdout and stderr
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	err = serverCmd.Start()
	if err != nil {
		log.Printf("Error starting server: %v", err)
		return
	}

	// Wait a bit for the server to start
	time.Sleep(1 * time.Second)

	log.Println("Outrig server started")

	// Monitor the server process in a goroutine
	go func(cmd *exec.Cmd, stdinPipe io.WriteCloser) {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Server process exited with error: %v", err)
		} else {
			log.Printf("Server process exited normally")
		}

		// Close stdin pipe
		stdinPipe.Close()
	}(serverCmd, stdin)
}

// stopServer stops the Outrig server
func stopServer() {
	serverLock.Lock()
	defer serverLock.Unlock()

	log.Println("Stopping Outrig server...")

	if serverCmd != nil && serverCmd.Process != nil {
		// Send interrupt signal to the server
		err := serverCmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("Error sending interrupt signal: %v", err)
			// Try to kill the process if interrupt fails
			err = serverCmd.Process.Kill()
			if err != nil {
				log.Printf("Error killing process: %v", err)
			}
		}

		// Wait for the process to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			_, err := serverCmd.Process.Wait()
			done <- err
		}()

		// Wait for process to exit or timeout
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Error waiting for process to exit: %v", err)
			}
		case <-time.After(5 * time.Second):
			log.Println("Timeout waiting for server to exit, forcing kill")
			serverCmd.Process.Kill()
		}

		serverCmd = nil
	}

	log.Println("Outrig server stopped")
}

// restartServer restarts the Outrig server
func restartServer() {
	log.Println("Restarting Outrig server...")

	// Stop and start the server
	stopServer()
	startServer()

	log.Println("Outrig server restarted")
}

func main() {
	// Set up logging
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "outrigapp.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("Starting OutrigApp")

	// Start the systray
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set up the systray icon and tooltip
	systray.SetIcon(iconData)
	systray.SetTooltip("Outrig")

	// Create menu items
	mOpen := systray.AddMenuItem("Open Outrig", "Open the Outrig web interface")
	systray.AddSeparator()
	mRestart := systray.AddMenuItem("Restart Server", "Restart the Outrig server")
	mQuit := systray.AddMenuItem("Quit Completely", "Quit the Application and Stop the Outrig Server")

	// Start the server immediately
	startServer()

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://localhost:5005")
			case <-mRestart.ClickedCh:
				restartServer()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("Exiting OutrigApp...")

	// Stop the server
	stopServer()

	log.Println("OutrigApp exited")
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("Error opening browser: %v\n", err)
	}
}
