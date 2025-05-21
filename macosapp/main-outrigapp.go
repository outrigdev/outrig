package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"time"

	"fyne.io/systray"
)

var (
	// Server process
	serverCmd          *exec.Cmd
	serverLock         sync.Mutex
	isFirstStart       = true
	serverFirstStartCh = make(chan bool)
	serverStartOnce    sync.Once

	statusUpdateLock sync.Mutex
	serverRunning    = false
	currentAppRuns   []TrayAppRunInfo
)

//go:embed assets/outrigapp-trayicon.png
var baseIconData []byte

//go:embed assets/outrigapp-trayicon-error.png
var errorIconData []byte

//go:embed assets/outrigapp-trayicon-conn.png
var connIconData []byte

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

// StatusResponse represents the response from the status endpoint
type StatusResponse struct {
	Success bool       `json:"success"`
	Data    StatusData `json:"data"`
}

type StatusData struct {
	Status         string           `json:"status"`
	Time           int64            `json:"time"`
	HasConnections bool             `json:"hasconnections"`
	AppRuns        []TrayAppRunInfo `json:"appruns"`
}

type TrayAppRunInfo struct {
	AppRunId  string `json:"apprunid"`
	AppName   string `json:"appname"`
	IsRunning bool   `json:"isrunning"`
	StartTime int64  `json:"starttime"`
}

// AppGroup represents a group of app runs with the same app name
type AppGroup struct {
	AppName string
	AppRuns []TrayAppRunInfo
}

// GetTopAppRun returns the highest ranked app run in the group
// Ranking is: IsRunning (true first), then StartTime (newest first)
func (g *AppGroup) GetTopAppRun() TrayAppRunInfo {
	if len(g.AppRuns) == 0 {
		return TrayAppRunInfo{}
	}

	// Sort the app runs
	sortAppRuns(g.AppRuns)

	// Return the first (highest ranked) app run
	return g.AppRuns[0]
}

// sortAppRuns sorts app runs by IsRunning (true first) and then by StartTime (newest first)
func sortAppRuns(appRuns []TrayAppRunInfo) {
	sort.Slice(appRuns, func(i, j int) bool {
		// Primary sort: IsRunning (true first)
		if appRuns[i].IsRunning != appRuns[j].IsRunning {
			return appRuns[i].IsRunning
		}

		// Secondary sort: StartTime (newest first)
		return appRuns[i].StartTime > appRuns[j].StartTime
	})
}

// ServerStatus holds the current status of the server
type ServerStatus struct {
	Running        bool
	HasConnections bool
	AppRuns        []TrayAppRunInfo
}

// getServerStatus checks if the server is running and responding
// Returns a ServerStatus struct with information about the server
func getServerStatus() ServerStatus {
	status := ServerStatus{
		Running:        false,
		HasConnections: false,
		AppRuns:        []TrayAppRunInfo{},
	}

	// Check if the server process exists
	if serverCmd == nil || serverCmd.Process == nil {
		return status
	}

	// Try to connect to the server
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	resp, err := client.Get("http://localhost:5005/api/status")
	if err != nil {
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return status
	}

	// Server is running
	status.Running = true

	// Try to parse the response
	var statusResp StatusResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&statusResp); err == nil {
		status.HasConnections = statusResp.Data.HasConnections
		status.AppRuns = statusResp.Data.AppRuns
	}

	return status
}

// updateServerStatus updates the icon and menu based on the server status
func updateServerStatus(serverStatus ServerStatus) {
	statusUpdateLock.Lock()
	defer statusUpdateLock.Unlock()

	wasRunning := serverRunning
	serverRunning = serverStatus.Running

	// Update icon if status changed
	if wasRunning != serverRunning {
		if serverRunning {
			// If there are active connections, use the connection icon
			if serverStatus.HasConnections {
				systray.SetIcon(connIconData)
				log.Printf("Server is running with connections, updated icon to connection\n")
			} else {
				systray.SetIcon(baseIconData)
				log.Printf("Server is running, updated icon to normal\n")
			}

			// Signal first start exactly once when server is running
			serverStartOnce.Do(func() {
				close(serverFirstStartCh)
			})
		} else {
			systray.SetIcon(errorIconData)
			log.Printf("Server is not running, updated icon to error\n")
		}
	} else if serverRunning {
		// Even if running status didn't change, update the icon based on connections
		if serverStatus.HasConnections {
			systray.SetIcon(connIconData)
		} else {
			systray.SetIcon(baseIconData)
		}
	}

	// Update menu with current app runs
	updateMenuWithAppRuns(serverStatus.AppRuns)
}

// startServer starts the Outrig server
func startServer() {
	serverLock.Lock()
	defer serverLock.Unlock()

	log.Printf("Starting Outrig server...\n")

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

	log.Printf("Outrig server started\n")

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

		// Update server status when process exits
		status := getServerStatus()
		updateServerStatus(status)
	}(serverCmd, stdin)
}

// stopServer stops the Outrig server
func stopServer() {
	serverLock.Lock()
	defer serverLock.Unlock()

	log.Printf("Stopping Outrig server...\n")

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
			log.Printf("Timeout waiting for server to exit, forcing kill\n")
			serverCmd.Process.Kill()
		}

		serverCmd = nil
	}

	log.Printf("Outrig server stopped\n")
}

// restartServer restarts the Outrig server
func restartServer() {
	log.Printf("Restarting Outrig server...\n")

	// Stop and start the server
	stopServer()
	startServer()

	log.Printf("Outrig server restarted\n")
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

// groupAppRuns groups app runs by app name and sorts each group
func groupAppRuns(appRuns []TrayAppRunInfo) []AppGroup {
	// Group app runs by app name
	groupMap := make(map[string][]TrayAppRunInfo)
	for _, appRun := range appRuns {
		groupMap[appRun.AppName] = append(groupMap[appRun.AppName], appRun)
	}

	// Convert map to slice of AppGroup
	groups := make([]AppGroup, 0, len(groupMap))
	for appName, runs := range groupMap {
		groups = append(groups, AppGroup{
			AppName: appName,
			AppRuns: runs,
		})
	}

	// Sort each group internally
	for i := range groups {
		sortAppRuns(groups[i].AppRuns)
	}

	// Sort the groups by their top app run
	sort.Slice(groups, func(i, j int) bool {
		topI := groups[i].GetTopAppRun()
		topJ := groups[j].GetTopAppRun()

		// Primary sort: IsRunning (true first)
		if topI.IsRunning != topJ.IsRunning {
			return topI.IsRunning
		}

		// Secondary sort: StartTime (newest first)
		return topI.StartTime > topJ.StartTime
	})

	return groups
}

// rebuildMenu completely rebuilds the systray menu
func rebuildMenu() {

	// Reset the entire menu
	systray.ResetMenu()

	// Add the main menu items
	mOpen := systray.AddMenuItem("Open Outrig", "Open the Outrig web interface")
	go func() {
		for range mOpen.ClickedCh {
			openBrowser("http://localhost:5005")
		}
	}()

	systray.AddSeparator()

	// Add the apps header
	mAppsHeader := systray.AddMenuItem("Recent Applications", "")
	mAppsHeader.Disable()

	// Add app run menu items if there are any
	if len(currentAppRuns) > 0 {
		// Group and sort app runs
		appGroups := groupAppRuns(currentAppRuns)

		// Add menu items for each app group
		for _, group := range appGroups {
			topRun := group.GetTopAppRun()

			// Create menu item text
			var menuText string
			if topRun.IsRunning {
				menuText = fmt.Sprintf("▶ %s", group.AppName)
			} else {
				menuText = fmt.Sprintf("⏹ %s", group.AppName)
			}

			// Add menu item
			menuItem := systray.AddMenuItem(menuText, fmt.Sprintf("Open %s logs", group.AppName))

			// Set up click handler
			go func(appRunId string) {
				for range menuItem.ClickedCh {
					url := fmt.Sprintf("http://localhost:5005/?appRunId=%s&tab=logs", appRunId)
					openBrowser(url)
				}
			}(topRun.AppRunId)
		}
	}

	systray.AddSeparator()

	// Add the control menu items
	mRestart := systray.AddMenuItem("Restart Server", "Restart the Outrig server")
	go func() {
		for range mRestart.ClickedCh {
			restartServer()
		}
	}()

	mQuit := systray.AddMenuItem("Quit Completely", "Quit the Application and Stop the Outrig Server")
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()
}

// updateMenuWithAppRuns updates the menu with the current app runs
func updateMenuWithAppRuns(appRuns []TrayAppRunInfo) {
	// Only update if app runs have changed
	if !reflect.DeepEqual(currentAppRuns, appRuns) {
		currentAppRuns = appRuns
		rebuildMenu()
	}
}

func onReady() {
	// Set up the systray icon and tooltip - start with error icon
	systray.SetIcon(errorIconData)
	systray.SetTooltip("Outrig")

	// Build the initial menu
	rebuildMenu()

	// Start the server immediately
	startServer()

	// Start a goroutine to monitor server status and update menu
	go func() {
		// Check server status every second
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			serverStatus := getServerStatus()
			updateServerStatus(serverStatus)
		}
	}()

	// Handle first start browser opening
	if isFirstStart {
		go func() {
			// Wait for server to be running
			select {
			case <-serverFirstStartCh:
				// Wait a bit more for the server to be fully ready
				time.Sleep(200 * time.Millisecond)
				if serverRunning {
					log.Printf("First start: opening browser\n")
					openBrowser("http://localhost:5005")
				}
			case <-time.After(10 * time.Second):
				// Timeout if server doesn't start
				log.Printf("Timeout waiting for server to start on first launch\n")
			}

			// No longer first start
			isFirstStart = false
		}()
	}
}

func onExit() {
	log.Printf("Exiting OutrigApp...\n")

	// Stop the server
	stopServer()

	log.Printf("OutrigApp exited\n")
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
