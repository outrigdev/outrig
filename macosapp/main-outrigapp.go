package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/Masterminds/semver/v3"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/updatecheck"
)

var (
	// Version information
	OutrigAppVersion = "v0.9.1"

	// Server process
	serverCmd          *exec.Cmd
	serverLock         sync.Mutex
	serverFirstStartCh = make(chan bool)
	serverStartOnce    sync.Once

	// Updater process
	globalUpdaterCmd *exec.Cmd
	updaterLock      sync.Mutex

	statusUpdateLock sync.Mutex
	rebuildMenuLock  sync.Mutex
	lastServerStatus ServerStatus
	lastIconType     string

	// Menu items
	mCheckUpdatesGlobal *systray.MenuItem

	isQuitting atomic.Bool
	isUserQuit atomic.Bool

	// CLI installation status
	isCliInstalled   atomic.Bool
	cliInstallFailed atomic.Bool

	// Updater status
	isUpdaterRunning atomic.Bool

	// Appcast update checking
	latestAppcastVersion string
	appcastVersionLock   sync.RWMutex
	lastAppcastCheck     atomic.Int64

	// Sparkle update checking
	lastSparkleCheck atomic.Int64
)

const (
	IconTypeTemplate = "template"
	IconTypeNormal   = "normal"
	IconTypeError    = "error"
	IconTypeConn     = "conn"

	// App status icon types
	IconTypeAppRunning = "app-running"
	IconTypeAppStopped = "app-stopped"

	// Log file names
	OutrigAppLogFile    = "outrigapp.log"
	OutrigServerLogFile = "outrigserver.log"

	// Update check interval
	AppcastUpdateCheckInterval = 8 * time.Hour
	SparkleUpdateCheckInterval = 8 * time.Hour
)

// LinkState represents the current state of the CLI symlink
type LinkState int8

const (
	LinkOK LinkState = iota
	LinkMissing
	LinkDangling
	LinkBadDest
	LinkClobber
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

//go:embed assets/wrench-template.png
var wrenchIconData []byte

//go:embed assets/download-template.png
var downloadIconData []byte

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

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func getLinkState(target, cliSource string) LinkState {
	fi, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return LinkMissing
	}
	if err != nil {
		return LinkClobber
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return LinkClobber
	}

	dest, _ := os.Readlink(target)
	if !filepath.IsAbs(dest) { // relative → make absolute
		dest = filepath.Join(filepath.Dir(target), dest)
	}
	dest = filepath.Clean(dest) // collapse “../”, “./” etc.

	switch {
	case dest == cliSource:
		return LinkOK
	case !pathExists(dest):
		return LinkDangling
	default:
		return LinkBadDest
	}
}

func randString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func atomicSymlink(src, dst string) error {
	tmp := filepath.Join(filepath.Dir(dst), ".outrig-tmp-"+randString(6))
	if err := os.Symlink(src, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func getCliPaths() (string, string) {
	execPath, _ := os.Executable()
	cliName := "outrig"
	cliSource := filepath.Join(filepath.Dir(execPath), cliName)
	target := filepath.Join("/usr/local/bin", cliName)
	if pathExists("/opt/homebrew/bin") {
		target = filepath.Join("/opt/homebrew/bin", cliName)
	}
	return cliSource, target
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
	if iconType != lastIconType {
		systray.SetTemplateIcon(iconDataMap[IconTypeTemplate], iconDataMap[IconTypeTemplate])
	}
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
	trayPid := os.Getpid()
	serverCmd = exec.Command(outrigPath, "monitor", "start", "--close-on-stdin", "--tray-pid", fmt.Sprintf("%d", trayPid))

	// Create a pipe for stdin
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		log.Printf("Error creating stdin pipe: %v", err)
		return
	}

	// We keep stdin open, but if outrigapp crashes, it will close automatically
	// causing the server to shut down due to the --close-on-stdin flag

	// Redirect server output to log file instead of console
	logPath := filepath.Join(os.TempDir(), OutrigServerLogFile)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Error opening log file for server output: %v", err)
		// Fall back to discarding output if we can't open log file
		serverCmd.Stdout = nil
		serverCmd.Stderr = nil
	} else {
		serverCmd.Stdout = logFile
		serverCmd.Stderr = logFile
	}

	err = serverCmd.Start()
	if err != nil {
		log.Printf("Error starting server: %v", err)
		return
	}

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

func stopUpdater() {
	updaterLock.Lock()
	defer updaterLock.Unlock()

	if globalUpdaterCmd == nil || globalUpdaterCmd.Process == nil {
		return
	}

	// Send interrupt signal to the updater
	err := globalUpdaterCmd.Process.Signal(os.Interrupt)
	if err != nil {
		// Try to kill the process if interrupt fails
		globalUpdaterCmd.Process.Kill()
	}

	globalUpdaterCmd = nil
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
	rebuildMenuLock.Lock()
	defer rebuildMenuLock.Unlock()

	appRuns := status.AppRuns

	// Reset the entire menu
	systray.ResetMenu()

	// Add the main menu items
	if status.Running {
		mOpen := systray.AddMenuItem("Open Outrig", "Open the Outrig web interface @ http://localhost:5005")
		go func() {
			for range mOpen.ClickedCh {
				err := utilfn.LaunchUrl("http://localhost:5005")
				if err != nil {
					log.Printf("Error opening browser: %v", err)
				}
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
					err := utilfn.LaunchUrl(url)
					if err != nil {
						log.Printf("Error opening browser: %v", err)
					}
				}
			}(topRun.AppRunId)
		}
	}

	systray.AddSeparator()
	addInstallCLIMenuItems(status)

	// Add version info
	if OutrigAppVersion != "" {
		versionItem := systray.AddMenuItem("Outrig "+OutrigAppVersion, "")
		versionItem.Disable() // Make it non-clickable
	}

	// Add check for updates menu item
	latestVersion := getLatestAppcastVersion()
	if isUpdaterRunning.Load() {
		mCheckUpdatesGlobal = systray.AddMenuItem("Updater Running (Focus Updater)...", "")
	} else if latestVersion != "" {
		mCheckUpdatesGlobal = systray.AddMenuItem("Install Outrig "+latestVersion+"...", "Install the latest version of Outrig")
		mCheckUpdatesGlobal.SetTemplateIcon(downloadIconData, downloadIconData)
	} else {
		mCheckUpdatesGlobal = systray.AddMenuItem("Check for Updates...", "")
	}
	go func() {
		for range mCheckUpdatesGlobal.ClickedCh {
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
			isUserQuit.Store(true)
			updateServerStatus(ServerStatus{})
			systray.Quit()
		}
	}()
}

func updateCheckUpdatesMenuItem() {
	rebuildMenuLock.Lock()
	defer rebuildMenuLock.Unlock()

	if mCheckUpdatesGlobal == nil {
		return
	}

	latestVersion := getLatestAppcastVersion()
	if isUpdaterRunning.Load() {
		mCheckUpdatesGlobal.SetTitle("Checking for updates...")
	} else if latestVersion != "" {
		mCheckUpdatesGlobal.SetTitle("Install Outrig " + latestVersion + "...")
		mCheckUpdatesGlobal.SetTemplateIcon(downloadIconData, downloadIconData)
		mCheckUpdatesGlobal.Enable()
	} else {
		mCheckUpdatesGlobal.SetTitle("Check for Updates...")
		mCheckUpdatesGlobal.Enable()
	}
}

func runServerStatusCheckLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		serverStatus := getServerStatus()
		updateServerStatus(serverStatus)
	}
}

func runAppcastUpdateCheckLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixMilli()
		lastCheck := lastAppcastCheck.Load()

		if now-lastCheck >= AppcastUpdateCheckInterval.Milliseconds() {
			checkAppcastUpdates()
		}
	}
}

func runBackgroundSparkleUpdaterLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixMilli()
		lastCheck := lastSparkleCheck.Load()

		if now-lastCheck >= SparkleUpdateCheckInterval.Milliseconds() {
			checkForUpdates(true)
		}
	}
}

func startServerOnStartup() {
	startServer()

	select {
	case <-serverFirstStartCh:
		time.Sleep(200 * time.Millisecond)
		log.Printf("Opening browser on startup\n")
		err := utilfn.LaunchUrl("http://localhost:5005")
		if err != nil {
			log.Printf("Error opening browser: %v", err)
		}
	case <-time.After(10 * time.Second):
		log.Printf("Timeout waiting for server to start on startup\n")
	}
}

func onReady() {
	updateIcon(IconTypeError)
	rebuildMenu(ServerStatus{})

	// Initialize sparkle check timestamp to prevent race condition
	lastSparkleCheck.Store(time.Now().UnixMilli())

	// Check for updates on startup
	go func() {
		checkForUpdates(true)
		checkAppcastUpdates()
	}()

	go runServerStatusCheckLoop()
	go runAppcastUpdateCheckLoop()
	go runBackgroundSparkleUpdaterLoop()
	go startServerOnStartup()
}

func onExit() {
	log.Printf("Exiting OutrigApp...\n")

	stopServer()
	if isUserQuit.Load() {
		stopUpdater()
	}

	log.Printf("OutrigApp exited\n")
}

func ensureCliLinkStartup() {
	cliSource, target := getCliPaths()
	state := getLinkState(target, cliSource)
	log.Printf("CLI link %s -> %s: %v", target, cliSource, state)
	switch state {
	case LinkMissing:
		_ = os.Symlink(cliSource, target)
	case LinkDangling:
		_ = os.Remove(target)
		_ = os.Symlink(cliSource, target)
	case LinkBadDest, LinkClobber:
		cliInstallFailed.Store(true)
	}
	newState := getLinkState(target, cliSource)
	log.Printf("CLI link %s -> %s: %v", target, cliSource, newState)
	isCliInstalled.Store(newState == LinkOK)
	if newState == LinkOK {
		cliInstallFailed.Store(false)
	}
}

func InstallOutrigCLI() {
	cliSource, targetPath := getCliPaths()
	state := getLinkState(targetPath, cliSource)
	log.Printf("CLI link %s -> %s: %v", targetPath, cliSource, state)
	var err error
	switch state {
	case LinkMissing:
		err = os.Symlink(cliSource, targetPath)
	case LinkDangling:
		_ = os.Remove(targetPath)
		err = os.Symlink(cliSource, targetPath)
	case LinkBadDest, LinkClobber:
		_ = os.RemoveAll(targetPath)
		err = atomicSymlink(cliSource, targetPath)
	}
	if err != nil {
		log.Printf("Error installing Outrig CLI: %v", err)
	}
	newState := getLinkState(targetPath, cliSource)
	log.Printf("CLI link %s -> %s: %v", targetPath, cliSource, newState)
	isCliInstalled.Store(newState == LinkOK)
	cliInstallFailed.Store(newState == LinkBadDest || newState == LinkClobber)
}

func addInstallCLIMenuItems(status ServerStatus) {
	cliSource, targetPath := getCliPaths()
	state := getLinkState(targetPath, cliSource)
	isCliInstalled.Store(state == LinkOK)
	cliInstallFailed.Store(state == LinkBadDest || state == LinkClobber)

	if state != LinkOK {
		var info, label string

		switch state {
		case LinkDangling:
			info = "Broken 'outrig' CLI link"
			label = "Repair CLI Link"
		case LinkBadDest:
			info = "Incorrect 'outrig' CLI link"
			label = "Repair CLI Link"
		case LinkClobber:
			info = "File named 'outrig' blocks CLI"
			label = "Overwrite with Outrig CLI"
		default: // LinkMissing
			label = "Install Outrig CLI"
		}

		if info != "" {
			item := systray.AddMenuItem(info, "")
			item.Disable()
		}

		mInstall := systray.AddMenuItem(label, "")
		mInstall.SetTemplateIcon(wrenchIconData, wrenchIconData)
		go func() {
			for range mInstall.ClickedCh {
				InstallOutrigCLI()
				rebuildMenu(status)
			}
		}()
		systray.AddSeparator()
	}
}

// IsOutrigCLIInstalled checks if the outrig CLI is installed in the system
func IsOutrigCLIInstalled() bool {
	cliSource, target := getCliPaths()
	return getLinkState(target, cliSource) == LinkOK
}

func foregroundUpdater() {
	exec.Command("osascript", "-e", `
		tell application id "run.outrig.Outrig"
			activate
		end tell
	`).Run()
}

// checkForUpdates launches the OutrigUpdater to check for updates
func checkForUpdates(first bool) {
	// Update the last check time first to be safe
	lastSparkleCheck.Store(time.Now().UnixMilli())

	// Test and set - only proceed if no updater is currently running
	if !isUpdaterRunning.CompareAndSwap(false, true) {
		log.Printf("Update check already in progress, bringing updater to foreground")
		foregroundUpdater()
		return
	}

	updateCheckUpdatesMenuItem()

	log.Printf("Checking for updates...\n")

	// Get the path to the OutrigUpdater
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
		isUpdaterRunning.Store(false)
		return
	}

	// Construct the path to the updater
	// For a macOS app bundle, the updater should be in the same directory as the main executable
	updaterPath := filepath.Join(filepath.Dir(execPath), "OutrigUpdater")

	// Check if the updater exists
	if _, err := os.Stat(updaterPath); os.IsNotExist(err) {
		log.Printf("Updater not found at %s: %v", updaterPath, err)
		isUpdaterRunning.Store(false)
		return
	}

	// Launch the updater
	var cmd *exec.Cmd
	pidStr := fmt.Sprintf("%d", os.Getpid())
	if first {
		cmd = exec.Command(updaterPath, "--first", "--pid", pidStr)
	} else {
		cmd = exec.Command(updaterPath, "--pid", pidStr)
	}

	// Redirect updater output to the same log file as OutrigApp
	logPath := filepath.Join(os.TempDir(), OutrigAppLogFile)
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
		isUpdaterRunning.Store(false)
		return
	}

	// Store the updater command reference
	updaterLock.Lock()
	globalUpdaterCmd = cmd
	updaterLock.Unlock()

	log.Printf("Update checker launched\n")

	// Monitor the updater process in a goroutine
	go func(cmdArg *exec.Cmd, logFileHandle *os.File) {
		defer func() {
			// Clear the updater command reference
			updaterLock.Lock()
			globalUpdaterCmd = nil
			updaterLock.Unlock()

			isUpdaterRunning.Store(false)
			updateCheckUpdatesMenuItem()
		}()

		err := cmdArg.Wait()
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

// checkAppcastUpdates checks for updates using the appcast and updates the menu if needed
func checkAppcastUpdates() {
	log.Printf("Checking appcast for updates...")

	// Get the latest version from appcast
	latestVersion, err := updatecheck.GetLatestAppcastRelease()
	if err != nil {
		log.Printf("Error checking appcast updates: %v", err)
		return
	}

	log.Printf("Latest appcast version: %s, current version: %s", latestVersion, OutrigAppVersion)

	// Compare versions using semver
	current, err := semver.NewVersion(OutrigAppVersion)
	if err != nil {
		log.Printf("Error parsing current version: %v", err)
		return
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		log.Printf("Error parsing latest version: %v", err)
		return
	}

	// Update the stored latest version
	appcastVersionLock.Lock()
	if latest.GreaterThan(current) {
		latestAppcastVersion = latestVersion
		log.Printf("New version available: %s", latestVersion)
	} else {
		latestAppcastVersion = ""
		log.Printf("No new version available")
	}
	appcastVersionLock.Unlock()

	// Update the last check time
	lastAppcastCheck.Store(time.Now().UnixMilli())

	// Update the menu item to reflect changes
	updateCheckUpdatesMenuItem()
}

// getLatestAppcastVersion returns the latest appcast version if available
func getLatestAppcastVersion() string {
	appcastVersionLock.RLock()
	defer appcastVersionLock.RUnlock()
	return latestAppcastVersion
}

func main() {
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), OutrigAppLogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// Set up signal handlers
	shutdownChan := make(chan os.Signal, 1)
	updateChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	signal.Notify(updateChan, syscall.SIGUSR1)

	// Handle shutdown signals
	go func() {
		sig := <-shutdownChan
		log.Printf("Received signal %v, shutting down gracefully", sig)
		isQuitting.Store(true)
		systray.Quit()
	}()

	// Handle SIGUSR1 for update checks
	go func() {
		for {
			sig := <-updateChan
			log.Printf("Received signal %v, checking for updates", sig)
			go checkForUpdates(false)
		}
	}()

	ensureCliLinkStartup()

	log.Printf("Starting OutrigApp")
	log.Printf("PATH: %s\n", os.Getenv("PATH"))
	log.Printf("CLI installed: %v\n", isCliInstalled.Load())

	systray.Run(onReady, onExit)
}
