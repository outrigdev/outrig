// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/collector/runtimestats"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"golang.org/x/term"
)

const ConnPollTime = 1 * time.Second
const MaxInternalLog = 100

type ControllerImpl struct {
	Lock                sync.Mutex // lock for this struct
	config              *config.Config
	pollerOnce          sync.Once                      // ensures poller is started only once
	AppInfo             ds.AppInfo                     // combined application information
	OutrigForceDisabled bool                           // whether outrig is force disabled
	Collectors          map[string]collector.Collector // map of collectors by name
	InternalLogBuf      *utilds.CirBuf[string]         // internal log for debugging
	transport           *Transport                     // handles connection management and packet sending
}

// this is idempotent
func MakeController(appName string, cfg config.Config) (*ControllerImpl, error) {
	if appName == "" {
		appName = determineAppName()
	}
	c := &ControllerImpl{
		Collectors:     make(map[string]collector.Collector),
		InternalLogBuf: utilds.MakeCirBuf[string](MaxInternalLog),
	}

	// Initialize transport
	c.transport = MakeTransport(&cfg)

	// Initialize AppInfo using the dedicated function
	c.AppInfo = c.createAppInfo(appName, &cfg)
	c.config = &cfg

	arCtx := ds.AppRunContext{
		AppRunId: c.AppInfo.AppRunId,
		IsDev:    c.config.Dev,
	}

	// Initialize collectors with their respective configurations
	logCollector := logprocess.GetInstance()
	logCollector.InitCollector(c, c.config.LogProcessorConfig, arCtx)
	c.Collectors[logCollector.CollectorName()] = logCollector

	goroutineCollector := goroutine.GetInstance()
	goroutineCollector.InitCollector(c, c.config.GoRoutineConfig, arCtx)
	c.Collectors[goroutineCollector.CollectorName()] = goroutineCollector

	watchCollector := watch.GetInstance()
	watchCollector.InitCollector(c, c.config.WatchConfig, arCtx)
	c.Collectors[watchCollector.CollectorName()] = watchCollector

	runtimeStatsCollector := runtimestats.GetInstance()
	runtimeStatsCollector.InitCollector(c, c.config.RuntimeStatsConfig, arCtx)
	c.Collectors[runtimeStatsCollector.CollectorName()] = runtimeStatsCollector

	return c, nil
}

func (c *ControllerImpl) InitialStart() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	// Check if Outrig is disabled via environment variable
	if os.Getenv(config.DisabledEnvName) != "" {
		c.OutrigForceDisabled = true
	}

	var connected bool
	if c.config.ConnectOnInit && !c.OutrigForceDisabled {
		connected, _ = c.connectInternal(true)
	}
	if connected {
		c.setEnabled(true)
	}
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig ConnPoller")
		c.runConnPoller()
	}()
}

// createAppInfo creates and initializes the AppInfo structure
func (c *ControllerImpl) createAppInfo(appName string, cfg *config.Config) ds.AppInfo {
	appInfo := ds.AppInfo{}

	// Initialize basic AppInfo
	appInfo.AppRunId = os.Getenv(base.AppRunIdEnvName)
	if appInfo.AppRunId == "" {
		appInfo.AppRunId = uuid.New().String()
	}
	if appName == "" {
		appName = determineAppName()
	}
	appInfo.AppName = appName

	// Set module name
	moduleName := cfg.ModuleName
	if moduleName == "" {
		moduleName = c.determineModuleName()
	}
	appInfo.ModuleName = moduleName

	// Initialize the rest of AppInfo
	appInfo.StartTime = time.Now().UnixMilli()
	appInfo.Args = utilfn.CopyStrArr(os.Args)
	appInfo.Executable, _ = os.Executable()
	appInfo.Env = utilfn.CopyStrArr(os.Environ())
	appInfo.Pid = os.Getpid()
	appInfo.OutrigSDKVersion = base.OutrigSDKVersion

	// Get user information
	user, err := user.Current()
	if err == nil {
		appInfo.User = user.Username
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		appInfo.Hostname = hostname
	}

	// Add build information
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		settings := make(map[string]string)
		for _, setting := range buildInfo.Settings {
			settings[setting.Key] = setting.Value
		}

		appInfo.BuildInfo = &ds.BuildInfoData{
			GoVersion: buildInfo.GoVersion,
			Path:      buildInfo.Path,
			Version:   buildInfo.Main.Version,
			Settings:  settings,
		}
	}

	return appInfo
}

// Connection management methods

func (c *ControllerImpl) WriteInitMessage(connected bool, connWrap *comm.ConnWrap, permErr error, transErr error) {
	if c.config.Quiet {
		return
	}

	brightCyan := "\x1b[96m"
	brightBlueUnderline := "\x1b[94;4m"
	reset := "\x1b[0m"

	if connected && connWrap != nil {
		printf("%s#outrig%s connected via %s (apprunid:%s)\n", brightCyan, reset, connWrap.PeerName, c.AppInfo.AppRunId)
		if connWrap.ServerResponse != nil && connWrap.ServerResponse.ServerHttpPort > 0 {
			printf("%s#outrig%s dashboard available @ %shttp://localhost:%d%s\n", brightCyan, reset, brightBlueUnderline, connWrap.ServerResponse.ServerHttpPort, reset)
		}
	} else if permErr != nil {
		printf("%s#outrig%s Permanent connection error: %v\n", brightCyan, reset, permErr)
		printf("%s#outrig%s Entering standby mode.\n", brightCyan, reset)
	} else if transErr != nil {
		if outrigPath, _ := exec.LookPath("outrig"); outrigPath == "" {
			printf("%s#outrig%s Outrig server not detected; entering standby mode. %sInfo: https://outrig.run%s\n", brightCyan, reset, brightBlueUnderline, reset)
		} else {
			printf("%s#outrig%s Outrig server installed but not running; entering standby mode.\n", brightCyan, reset)
		}
	} else {
		// shouldn't happen (we should either be connected or have an error)
	}
}

// lock should be held
// returns (connected, transientError)
func (c *ControllerImpl) connectInternal(init bool) (rtnConnected bool, rtnErr error) {
	// Check if already connected to prevent redundant connections
	if c.transport.HasConnections() {
		return false, nil
	}
	if c.OutrigForceDisabled {
		return false, nil
	}
	var connWrap *comm.ConnWrap
	var permErr, transErr error
	defer func() {
		if !init || c.config.Quiet {
			return
		}
		c.WriteInitMessage(rtnConnected, connWrap, permErr, transErr)
	}()

	// Connect using config which handles environment overrides
	connWrap, permErr, transErr = comm.Connect(comm.ConnectionModePacket, "", c.AppInfo.AppRunId, c.config)
	if transErr != nil {
		// Connection failed
		return false, transErr
	}
	if permErr != nil {
		c.OutrigForceDisabled = true
		if !c.config.Quiet {
			fmt.Printf("[outrig] connection error: %v\n", permErr)
		}
		return false, nil
	}
	// Connection and handshake successful
	if !c.config.Quiet && !init {
		fmt.Printf("[outrig] connected via %s, apprunid:%s\n", connWrap.PeerName, c.AppInfo.AppRunId)
	}
	c.transport.AddConn(connWrap)
	c.sendAppInfo()

	// Force a full goroutine update on the next dump after a new connection
	if goCollector, ok := c.Collectors["goroutine"].(*goroutine.GoroutineCollector); ok {
		goCollector.SetNextSendFull(true)
	}
	// Force the watch collector to send a full update on the next cycle as well
	if watchCollector, ok := c.Collectors["watch"].(*watch.WatchCollector); ok {
		watchCollector.SetNextSendFull(true)
	}

	return true, nil
}

// lock should be held
func (c *ControllerImpl) disconnectInternal_nolock() {
	c.transport.CloseAllConns()
	c.setEnabled(false)
}

func (c *ControllerImpl) Enable() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	// Check if Outrig is disabled via environment variable
	if os.Getenv(config.DisabledEnvName) != "" {
		// Don't allow enabling if the environment variable is set
		return
	}

	c.OutrigForceDisabled = false
	isConnected := c.transport.HasConnections()
	if !isConnected {
		isConnected, _ = c.connectInternal(false)
	}
	if isConnected {
		c.setEnabled(true)
	}
}

func (c *ControllerImpl) Disable(disconnect bool) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.OutrigForceDisabled = true
	c.setEnabled(false)
	if disconnect {
		c.disconnectInternal_nolock()
	}
}

// Configuration methods

func (c *ControllerImpl) IsForceDisabled() bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return c.OutrigForceDisabled
}

func (c *ControllerImpl) GetConfig() config.Config {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return *c.config
}

func (c *ControllerImpl) GetAppRunId() string {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return c.AppInfo.AppRunId
}

func (c *ControllerImpl) SendPacket(pk *ds.PacketType) (bool, error) {
	return c.transport.SendPacket(pk, false)
}

func (c *ControllerImpl) sendAppInfo() {
	appInfoPacket := &ds.PacketType{
		Type: ds.PacketTypeAppInfo,
		Data: &c.AppInfo,
	}
	c.transport.SendPacket(appInfoPacket, true)
}

func (c *ControllerImpl) sendCollectorStatus() {
	if !global.OutrigEnabled.Load() {
		// Don't send collector status if Outrig is disabled
		return
	}

	statuses := c.GetCollectorStatuses()
	collectorStatusPacket := &ds.PacketType{
		Type: ds.PacketTypeCollectorStatus,
		Data: statuses,
	}
	c.transport.SendPacket(collectorStatusPacket, false)
}

// Initialization methods

func (c *ControllerImpl) determineModuleName() string {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Keep going up until we find go.mod or hit the filesystem root
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found the go.mod file, now parse it
			content, err := os.ReadFile(goModPath)
			if err != nil {
				return ""
			}

			// Look for the module line
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					// Extract the module name
					moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					return moduleName
				}
			}
			return "" // Module declaration not found
		}

		// Move up to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root without finding go.mod
			break
		}
		dir = parent
	}

	return "" // No go.mod found
}

func determineAppName() string {
	execPath, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	return filepath.Base(execPath)
}

func (c *ControllerImpl) Shutdown() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	// TODO: wait for last log lines to be sent
	// TODO: send shutdown log lines
	c.disconnectInternal_nolock()
}

// Private methods

func (c *ControllerImpl) runConnPoller() {
	c.pollerOnce.Do(func() {
		for {
			c.pollConn()
			c.sendCollectorStatus()
			time.Sleep(ConnPollTime)
		}
	})
}

func (c *ControllerImpl) pollConn() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if c.transport.HasConnections() {
		// Send collector status when connected
		return
	}
	// Try to connect
	connected, _ := c.connectInternal(false)
	if connected {
		c.setEnabled(true)
		return
	}
	// No connections after trying to connect, so disable
	c.setEnabled(false)
}

// lock should be held
func (c *ControllerImpl) setEnabled(enabled bool) {
	oldEnabled := global.OutrigEnabled.Load()
	if enabled == oldEnabled {
		return
	}
	if enabled {
		global.OutrigEnabled.Store(true)
		// Only enable collectors that have Enabled set to true in their config
		for name, collector := range c.Collectors {
			switch name {
			case "logprocess":
				if c.config.LogProcessorConfig.Enabled {
					collector.Enable()
				}
			case "goroutine":
				if c.config.GoRoutineConfig.Enabled {
					collector.Enable()
				}
			case "watch":
				if c.config.WatchConfig.Enabled {
					collector.Enable()
				}
			case "runtimestats":
				if c.config.RuntimeStatsConfig.Enabled {
					collector.Enable()
				}
			}
		}
	} else {
		for _, collector := range c.Collectors {
			collector.Disable()
		}
		global.OutrigEnabled.Store(false)
	}
}

func (c *ControllerImpl) ILog(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if len(msg) == 0 {
		return
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	c.InternalLogBuf.Write(msg)
}

var (
	isStdoutTerminalOnce  sync.Once
	isStdoutTerminalValue bool
)

// isStdoutATerminal returns whether stdout is a terminal.
// The result is cached after the first call.
func isStdoutATerminal() bool {
	isStdoutTerminalOnce.Do(func() {
		// Use loginitex.OrigStdout() to get the original stdout file descriptor
		// This handles cases where stdout has been redirected by the Outrig log capture
		isStdoutTerminalValue = term.IsTerminal(int(loginitex.OrigStdout().Fd()))
	})
	return isStdoutTerminalValue
}

var ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*m")

// printf formats and prints a string to stdout, stripping ANSI escape sequences
// if stdout is not a terminal.
func printf(format string, args ...any) {
	formatted := fmt.Sprintf(format, args...)

	// If stdout is not a terminal, strip ANSI escape sequences
	if !isStdoutATerminal() {
		formatted = ansiRegex.ReplaceAllString(formatted, "")
	}

	fmt.Print(formatted)
}

// GetCollectorStatuses returns the status of all collectors
func (c *ControllerImpl) GetCollectorStatuses() map[string]ds.CollectorStatus {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	statuses := make(map[string]ds.CollectorStatus)
	for name, collector := range c.Collectors {
		statuses[name] = collector.GetStatus()
	}
	return statuses
}
