package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/collector/runtimestats"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second

type ControllerImpl struct {
	Lock                 sync.Mutex               // lock for this struct
	conn                 atomic.Pointer[net.Conn] // connection to server (atomic pointer for lock-free access)
	config               *ds.Config
	ClientAddr           string                         // client address
	pollerOnce           sync.Once                      // ensures poller is started only once
	AppInfo              ds.AppInfo                     // combined application information
	TransportErrors      int64                          // count of transport errors
	TransportPacketsSent int64                          // count of packets sent
	OutrigConnected      bool                           // whether outrig is connected
	OutrigForceDisabled  bool                           // whether outrig is force disabled
	Collectors           map[string]collector.Collector // map of collectors by name
}

func MakeController(config ds.Config) (*ControllerImpl, error) {
	c := &ControllerImpl{
		Collectors: make(map[string]collector.Collector),
	}

	// Initialize AppInfo using the dedicated function
	c.AppInfo = c.createAppInfo(&config)
	c.config = &config

	if !config.StartAsync {
		c.Connect()
	}

	// Initialize collectors
	logCollector := logprocess.GetInstance()
	logCollector.InitCollector(c)
	c.Collectors[logCollector.CollectorName()] = logCollector

	goroutineCollector := goroutine.GetInstance()
	goroutineCollector.InitCollector(c)
	c.Collectors[goroutineCollector.CollectorName()] = goroutineCollector

	watchCollector := watch.GetInstance()
	watchCollector.InitCollector(c)
	c.Collectors[watchCollector.CollectorName()] = watchCollector

	runtimeStatsCollector := runtimestats.GetInstance()
	runtimeStatsCollector.InitCollector(c)
	c.Collectors[runtimeStatsCollector.CollectorName()] = runtimeStatsCollector

	isEnabled := global.OutrigEnabled.Load()
	if isEnabled {
		for _, collector := range c.Collectors {
			collector.Enable()
		}
	}
	go c.runConnPoller()
	
	// Initialize crash output handling if enabled
	c.initCrashOutput()
	
	return c, nil
}

// createAppInfo creates and initializes the AppInfo structure
func (c *ControllerImpl) createAppInfo(config *ds.Config) ds.AppInfo {
	appInfo := ds.AppInfo{}

	// Initialize basic AppInfo
	appInfo.AppRunId = uuid.New().String()

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

func (c *ControllerImpl) Connect() bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	if c.OutrigForceDisabled {
		return false
	}
	if global.OutrigEnabled.Load() {
		return false
	}

	atomic.StoreInt64(&c.TransportErrors, 0)
	var conn net.Conn
	var err error

	// Attempt domain socket if not disabled
	if c.config.DomainSocketPath != "-" {
		dsPath := utilfn.ExpandHomeDir(c.config.DomainSocketPath)
		if _, errStat := os.Stat(dsPath); errStat == nil {
			conn, err = net.DialTimeout("unix", dsPath, 2*time.Second)
			if err == nil {
				fmt.Printf("Outrig connected via domain socket: %v\n", dsPath)
				c.conn.Store(&conn)
				c.ClientAddr = c.config.DomainSocketPath
				c.sendAppInfo()
				c.setEnabled(true)
				c.OutrigConnected = true
				return true
			}
		}
	}

	// Fall back to TCP if not disabled
	if c.config.ServerAddr != "-" {
		conn, err = net.DialTimeout("tcp", c.config.ServerAddr, 2*time.Second)
		if err == nil {
			fmt.Printf("Outrig connected via TCP: %v\n", c.config.ServerAddr)
			c.conn.Store(&conn)
			c.ClientAddr = c.config.ServerAddr
			c.sendAppInfo()
			c.setEnabled(true)
			c.OutrigConnected = true
			return true
		}
	}

	return false
}

func (c *ControllerImpl) Disconnect() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fmt.Printf("Outrig disconnected from %s\n", c.ClientAddr)
	c.OutrigConnected = false
	c.setEnabled(false)

	connPtr := c.conn.Load()
	if connPtr == nil {
		return
	}

	conn := *connPtr
	c.conn.Store(nil)
	time.Sleep(50 * time.Millisecond)
	conn.Close()
}

func (c *ControllerImpl) Enable() {
	c.Lock.Lock()
	c.OutrigForceDisabled = false
	if c.OutrigConnected {
		c.setEnabled(true)
	}
	c.Lock.Unlock()
	c.Connect()
}

func (c *ControllerImpl) Disable(disconnect bool) {
	c.Lock.Lock()
	c.OutrigForceDisabled = true
	c.Lock.Unlock()
	c.setEnabled(false)

	if disconnect {
		c.Disconnect()
	}
}

// Configuration methods

func (c *ControllerImpl) GetConfig() ds.Config {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return *c.config
}

// Transport methods

func (c *ControllerImpl) sendPacketInternal(pk *ds.PacketType) (bool, error) {
	// No lock needed - using atomic pointer
	connPtr := c.conn.Load()
	if connPtr == nil {
		return false, nil
	}
	conn := *connPtr

	barr, err := json.Marshal(pk)
	if err != nil {
		return false, err
	}

	barr = append(barr, '\n')
	_, err = conn.Write(barr)
	if err != nil {
		atomic.AddInt64(&c.TransportErrors, 1) // this will force a disconnect later
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
	// TODO: wait for last log lines to be sent
	// TODO: send shutdown log lines
	c.Disconnect()
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
	enabled := global.OutrigEnabled.Load()
	if enabled {
		// check for errors
		if atomic.LoadInt64(&c.TransportErrors) > 0 {
			c.Disconnect()
			return
		}
		return
	} else {
		c.Connect()
	}
}

func (c *ControllerImpl) setEnabled(enabled bool) {
	oldEnabled := global.OutrigEnabled.Load()
	if enabled == oldEnabled {
		return
	}
	if enabled {
		global.OutrigEnabled.Store(true)
		go func() {
			for _, collector := range c.Collectors {
				collector.Enable()
			}
		}()
	} else {
		go func() {
			for _, collector := range c.Collectors {
				collector.Disable()
			}
		}()
		global.OutrigEnabled.Store(false)
	}
}
