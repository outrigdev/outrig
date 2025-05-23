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
	"sync/atomic"
	"time"

	"fyne.io/systray"
)

var (
	// Version information
	OutrigAppVersion = "v0.5.13"

	// Server process
	serverCmd          *exec.Cmd
	serverLock         sync.Mutex
	isFirstStart       = true
	serverFirstStartCh = make(chan bool)
	serverStartOnce    sync.Once

	statusUpdateLock sync.Mutex
	lastServerStatus ServerStatus
	lastIconType     string

	isQuitting atomic.Bool

	// CLI installation status
	isCliInstalled   atomic.Bool
	cliInstallFailed atomic.Bool
)

const (
	IconTypeTemplate = "template"
	IconTypeNormal   = "normal"
	IconTypeError    = "error"
	IconTypeConn     = "conn"

	// App status icon types
	IconTypeAppRunning = "app-running"
	IconTypeAppStopped = "app-stopped"
)

var iconDataMap = make(map[string][]byte)

//go:embed assets/outrigapp-trayicon.png
var baseIconData []byte

//go:embed assets/outrigapp-trayicon-error.png
var errorIconData []byte

//go:embed assets/outrigapp-trayicon-conn.png
var connIconData []byte

//go:embed assets/wifi-template.png
var wifiIconData []byte

//go:embed assets/wifioff-template.png
var wifiOffIconData []byte

//go:embed assets/outrigapp-trayicon-template.png
var trayIconTemplateData []byte

func init() {
	iconDataMap[IconTypeTemplate] = trayIconTemplateData
	iconDataMap[IconTypeNormal] = baseIconData
	iconDataMap[IconTypeError] = errorIconData
	iconDataMap[IconTypeConn] = connIconData
	iconDataMap[IconTypeAppRunning] = wifiIconData
	iconDataMap[IconTypeAppStopped] = wifiOffIconData
}

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
	Version        string           `json:"version"`
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
	Version        string
}

func getIconTypeForStatus(status ServerStatus) string {
	if !status.Running {
		return IconTypeError
	}
	if status.HasConnections {
		return IconTypeConn
	}
	return IconTypeNormal
}

func updateIcon(iconType string) {
	if iconType == lastIconType {
		return
	}
	systray.SetTemplateIcon(iconDataMap[IconTypeTemplate], iconDataMap[IconTypeTemplate])
	var statusMsg string
	switch iconType {
	case IconTypeNormal:
		statusMsg = "Outrig Server is Running"
	case IconTypeConn:
		statusMsg = "Outrig Server is running with Active Connections"
	case IconTypeError:
		statusMsg = "Server is Not Running"
	}
	systray.SetTooltip(statusMsg)
	lastIconType = iconType
}

func getServerStatus() ServerStatus {
	status := ServerStatus{
		Running:        false,
		HasConnections: false,
		AppRuns:        []TrayAppRunInfo{},
		Version:        "",
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

	status.Running = true
	// Try to parse the response
	var statusResp StatusResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&statusResp); err == nil {
		status.HasConnections = statusResp.Data.HasConnections
		status.AppRuns = statusResp.Data.AppRuns
		status.Version = statusResp.Data.Version

		// Sort AppRuns by apprunid to ensure consistent ordering
		sort.Slice(status.AppRuns, func(i, j int) bool {
			return status.AppRuns[i].AppRunId < status.AppRuns[j].AppRunId
		})
	}

	return status
}

func updateServerStatus(serverStatus ServerStatus) {
	statusUpdateLock.Lock()
	defer statusUpdateLock.Unlock()

	if isQuitting.Load() {
		updateIcon(IconTypeError)
		return
	}

	defer func() {
		lastServerStatus = serverStatus
	}()

	iconType := getIconTypeForStatus(serverStatus)
	updateIcon(iconType)

	if serverStatus.Running {
		serverStartOnce.Do(func() {
			close(serverFirstStartCh)
		})
	}

	if !reflect.DeepEqual(lastServerStatus, serverStatus) {
		rebuildMenu(serverStatus)
	}
}

func startServer() {
	serverLock.Lock()
	defer serverLock.Unlock()

	log.Printf("Starting Outrig server...\n")
	outrigPath := getOutrigPath()
	serverCmd = exec.Command(outrigPath, "server", "--close-on-stdin")

	// Create a pipe for stdin
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		log.Printf("Error creating stdin pipe: %v", err)
		return
	}

	// We keep stdin open, but if outrigapp crashes, it will close automatically
	// causing the server to shut down due to the --close-on-stdin flag

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

func restartServer() {
	log.Printf("Restarting Outrig server...\n")

	stopServer()
	startServer()

	log.Printf("Outrig server restarted\n")
}

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

func rebuildMenu(status ServerStatus) {
	appRuns := status.AppRuns

	// Reset the entire menu
	systray.ResetMenu()

	// Add the main menu items
	if status.Running {
		mOpen := systray.AddMenuItem("Open Outrig", "Open the Outrig web interface @ http://localhost:5005")
		go func() {
			for range mOpen.ClickedCh {
				openBrowser("http://localhost:5005")
			}
		}()
	} else {
		mNotRunning := systray.AddMenuItem("Outrig Server Not Running", "")
		mNotRunning.Disable()
	}

	systray.AddSeparator()

	// Add the apps header
	mAppsHeader := systray.AddMenuItem("Recent Applications", "")
	mAppsHeader.Disable()

	// Add app run menu items if there are any
	if len(appRuns) > 0 {
		// Group and sort app runs
		appGroups := groupAppRuns(appRuns)

		// Add menu items for each app group
		for _, group := range appGroups {
			topRun := group.GetTopAppRun()

			// Create menu item text
			var menuText string
			var iconType string

			if topRun.IsRunning {
				menuText = group.AppName
				iconType = IconTypeAppRunning
			} else {
				menuText = group.AppName
				iconType = IconTypeAppStopped
			}

			// Add menu item
			menuItem := systray.AddMenuItem(menuText, fmt.Sprintf("Open '%s' App Run in the Outrig web interface", group.AppName))

			// Set the icon for the menu item
			menuItem.SetTemplateIcon(iconDataMap[iconType], iconDataMap[iconType])

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

	// Add CLI installation menu item if needed
	if !isCliInstalled.Load() {
		if cliInstallFailed.Load() {
			mInstallFailed := systray.AddMenuItem("Outrig CLI Installation Failed", "")
			mInstallFailed.Disable()
			systray.AddSeparator()
		}
		mInstallCli := systray.AddMenuItem("Install 'outrig' CLI Command...", "Install the outrig CLI command for system-wide use")
		go func() {
			for range mInstallCli.ClickedCh {
				InstallOutrigCLI()
				rebuildMenu(status)
			}
		}()
		systray.AddSeparator()
	}

	// Add version info
	if OutrigAppVersion != "" {
		versionItem := systray.AddMenuItem("Outrig "+OutrigAppVersion, "")
		versionItem.Disable() // Make it non-clickable
	}

	// Add check for updates menu item
	mCheckUpdates := systray.AddMenuItem("Check for Updates...", "")
	go func() {
		for range mCheckUpdates.ClickedCh {
			checkForUpdates(false)
		}
	}()

	systray.AddSeparator()

	mRestart := systray.AddMenuItem("Restart Outrig Server", "")
	go func() {
		for range mRestart.ClickedCh {
			restartServer()
		}
	}()

	mQuit := systray.AddMenuItem("Quit Outrig Completely", "")
	go func() {
		for range mQuit.ClickedCh {
			isQuitting.Store(true)
			updateServerStatus(ServerStatus{})
			systray.Quit()
		}
	}()
}

func onReady() {
	updateIcon(IconTypeError)
	rebuildMenu(ServerStatus{})

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
				// Wait a bit more to make sure the server is ready to accept connections
				time.Sleep(200 * time.Millisecond)
				log.Printf("First start: opening browser\n")
				openBrowser("http://localhost:5005")
			case <-time.After(10 * time.Second):
				// Timeout if server doesn't start
				log.Printf("Timeout waiting for server to start on first launch\n")
			}
			// No longer first start
			isFirstStart = false
		}()
	}

	// Check for updates in background on startup
	go func() {
		// Wait a bit after startup before checking for updates
		time.Sleep(5 * time.Second)
		checkForUpdates(true)
	}()
}

func onExit() {
	log.Printf("Exiting OutrigApp...\n")

	stopServer()

	log.Printf("OutrigApp exited\n")
}

func InstallOutrigCLI() {
	err := installOutrigCLIInternal()
	if err != nil {
		log.Printf("Error installing Outrig CLI: %v", err)
		cliInstallFailed.Store(true)
		return
	}
	isCliInstalled.Store(true)
}

func installOutrigCLIInternal() error {
	appPath, err := os.Executable()
	if err != nil {
		return err
	}
	cliName := "outrig"
	cliSource := filepath.Join(filepath.Dir(appPath), cliName)

	targets := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	var targetPath string
	for _, dir := range targets {
		if isDirectoryWriteable(dir) {
			targetPath = filepath.Join(dir, cliName)
			return os.Symlink(cliSource, targetPath)
		}
	}

	// Fall back to osascript (GUI admin prompt)
	targetPath = filepath.Join("/usr/local/bin", cliName)
	msgStr := "Outrig needs to link its CLI command (outrig) to /usr/local/bin to enable automatic log capturing."
	script := fmt.Sprintf(`do shell script "ln -sf '%s' '%s'" with administrator privileges with prompt "%s"`, cliSource, targetPath, msgStr)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func isDirectoryWriteable(path string) bool {
	testFile := filepath.Join(path, ".tmp_test_write")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		return false
	}
	os.Remove(testFile)
	return true
}

// IsOutrigCLIInstalled checks if the outrig CLI is installed in the system
func IsOutrigCLIInstalled() bool {
	// First try LookPath to check if it's in PATH
	_, err := exec.LookPath("outrig")
	if err == nil {
		return true
	}

	// If LookPath fails, check common installation paths
	cliPaths := []string{
		"/opt/homebrew/bin/outrig",
		"/usr/local/bin/outrig",
	}

	for _, path := range cliPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

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

// checkForUpdates launches the OutrigUpdater to check for updates
func checkForUpdates(background bool) {
	if background {
		log.Printf("Checking for updates in background...\n")
	} else {
		log.Printf("Checking for updates...\n")
	}

	// Get the path to the OutrigUpdater
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
		return
	}

	// Construct the path to the updater
	// For a macOS app bundle, the updater should be in the same directory as the main executable
	updaterPath := filepath.Join(filepath.Dir(execPath), "OutrigUpdater")

	// Check if the updater exists
	if _, err := os.Stat(updaterPath); os.IsNotExist(err) {
		log.Printf("Updater not found at %s: %v", updaterPath, err)
		return
	}

	// Launch the updater
	var cmd *exec.Cmd
	if background {
		cmd = exec.Command(updaterPath, "--background")
	} else {
		cmd = exec.Command(updaterPath)
	}
	
	// Redirect updater output to the same log file as OutrigApp
	logPath := filepath.Join(os.TempDir(), "outrigapp.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	
	err = cmd.Start()
	if err != nil {
		log.Printf("Error launching updater: %v", err)
		if logFile != nil {
			logFile.Close()
		}
		return
	}

	log.Printf("Update checker launched\n")

	// Monitor the updater process in a goroutine
	go func(updaterCmd *exec.Cmd, logFileHandle *os.File) {
		err := updaterCmd.Wait()
		if err != nil {
			log.Printf("Update checker exited with error: %v", err)
		} else {
			log.Printf("Update checker completed successfully")
		}

		// Close log file handle
		if logFileHandle != nil {
			logFileHandle.Close()
		}
	}(cmd, logFile)
}

func main() {
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "outrigapp.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// Check if CLI is installed
	if IsOutrigCLIInstalled() {
		isCliInstalled.Store(true)
	} else {
		InstallOutrigCLI()
	}

	log.Printf("Starting OutrigApp")
	log.Printf("PATH: %s\n", os.Getenv("PATH"))
	log.Printf("CLI installed: %v\n", isCliInstalled.Load())

	systray.Run(onReady, onExit)
}
