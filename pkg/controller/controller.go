// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/collector/runtimestats"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second
const MaxInternalLog = 100

type ControllerImpl struct {
	Lock                 sync.Mutex // lock for this struct
	config               *ds.Config
	pollerOnce           sync.Once                      // ensures poller is started only once
	AppInfo              ds.AppInfo                     // combined application information
	TransportPacketsSent int64                          // count of packets sent
	OutrigForceDisabled  bool                           // whether outrig is force disabled
	Collectors           map[string]collector.Collector // map of collectors by name
	InternalLogBuf       *utilds.CirBuf[string]         // internal log for debugging

	connLock sync.Mutex
	conn     *comm.ConnWrap // connection to server
}

// this is idempotent
func MakeController(config ds.Config) (*ControllerImpl, error) {
	c := &ControllerImpl{
		Collectors:     make(map[string]collector.Collector),
		InternalLogBuf: utilds.MakeCirBuf[string](MaxInternalLog),
	}

	// Initialize AppInfo using the dedicated function
	c.AppInfo = c.createAppInfo(&config)
	c.config = &config

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
	if os.Getenv(ds.DisabledEnvName) != "" {
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
func (c *ControllerImpl) createAppInfo(config *ds.Config) ds.AppInfo {
	appInfo := ds.AppInfo{}

	// Initialize basic AppInfo
	appInfo.AppRunId = os.Getenv(base.AppRunIdEnvName)
	if appInfo.AppRunId == "" {
		appInfo.AppRunId = uuid.New().String()
	}

	// Set app name
	appName := config.AppName
	if appName == "" {
		appName = c.determineAppName()
	}
	appInfo.AppName = appName

	// Set module name
	moduleName := config.ModuleName
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

// lock should be held
// returns (connected, transientError)
func (c *ControllerImpl) connectInternal(init bool) (rtnConnected bool, rtnErr error) {
	// Check if already connected to prevent redundant connections
	if c.isConnected() {
		return false, nil
	}
	if c.OutrigForceDisabled {
		return false, nil
	}
	defer func() {
		if !init {
			return
		}
	}()

	// Check for domain socket override from environment variable
	domainSocketPath := c.config.DomainSocketPath
	if envPath := os.Getenv(ds.DomainSocketEnvName); envPath != "" {
		domainSocketPath = envPath
	}

	// Use the new Connect function to establish a connection
	connWrap, permErr, transErr := comm.Connect(comm.ConnectionModePacket, "", c.AppInfo.AppRunId, domainSocketPath, "")
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
	if !c.config.Quiet {
		fmt.Printf("[outrig] connected via %s, apprunid:%s\n", connWrap.PeerName, c.AppInfo.AppRunId)
	}
	c.setConn(connWrap)
	c.sendAppInfo()
	return true, nil
}

func (c *ControllerImpl) isConnected() bool {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	return c.conn != nil
}

func (c *ControllerImpl) setConn(conn *comm.ConnWrap) {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	c.conn = conn
}

func (c *ControllerImpl) closeConn_nolock() {
	if c.conn == nil {
		return
	}
	if !c.config.Quiet {
		fmt.Printf("[outrig] disconnecting from %s\n", c.conn.PeerName)
	}
	c.conn.Close()
	c.conn = nil
}

func (c *ControllerImpl) closeConn() {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	c.closeConn_nolock()
}

// lock should be held
func (c *ControllerImpl) disconnectInternal() {
	c.closeConn()
	c.setEnabled(false)
}

func (c *ControllerImpl) Enable() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	// Check if Outrig is disabled via environment variable
	if os.Getenv(ds.DisabledEnvName) != "" {
		// Don't allow enabling if the environment variable is set
		return
	}

	c.OutrigForceDisabled = false
	isConnected := c.isConnected()
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
		c.disconnectInternal()
	}
}

// Configuration methods

func (c *ControllerImpl) IsForceDisabled() bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return c.OutrigForceDisabled
}

func (c *ControllerImpl) GetConfig() ds.Config {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return *c.config
}

func (c *ControllerImpl) GetAppRunId() string {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return c.AppInfo.AppRunId
}

// Transport methods

func (c *ControllerImpl) sendPacketInternal(pk *ds.PacketType) (bool, error) {
	barr, err := json.Marshal(pk)
	if err != nil {
		return false, err
	}
	jsonStr := string(barr)
	c.connLock.Lock()
	defer c.connLock.Unlock()
	if c.conn == nil {
		return false, nil
	}
	err = c.conn.WriteLine(jsonStr)
	if err != nil {
		c.ILog("[error] writing to %s: %v\n", c.conn.PeerName, err)
		c.closeConn_nolock()
		go func() {
			ioutrig.I.SetGoRoutineName("#outrig sendPacket:error")
			c.Lock.Lock()
			defer c.Lock.Unlock()
			c.disconnectInternal()
		}()
		return false, nil
	}
	atomic.AddInt64(&c.TransportPacketsSent, 1)
	return true, nil
}

func (c *ControllerImpl) SendPacket(pk *ds.PacketType) (bool, error) {
	if !global.OutrigEnabled.Load() {
		return false, nil
	}

	return c.sendPacketInternal(pk)
}

func (c *ControllerImpl) sendAppInfo() {
	// Send AppInfo as the first packet
	appInfoPacket := &ds.PacketType{
		Type: ds.PacketTypeAppInfo,
		Data: &c.AppInfo,
	}
	c.sendPacketInternal(appInfoPacket)
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

func (c *ControllerImpl) determineAppName() string {
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
	c.disconnectInternal()
}

// Private methods

func (c *ControllerImpl) runConnPoller() {
	c.pollerOnce.Do(func() {
		for {
			c.pollConn()
			time.Sleep(ConnPollTime)
		}
	})
}

func (c *ControllerImpl) pollConn() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if c.isConnected() {
		return
	}
	connected, _ := c.connectInternal(false)
	if connected {
		c.setEnabled(true)
	}
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
