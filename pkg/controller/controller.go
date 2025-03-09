package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second

type ControllerImpl struct {
	Lock       sync.Mutex                // lock for this struct
	conn       atomic.Pointer[net.Conn]  // connection to server (atomic pointer for lock-free access)
	configPtr  atomic.Pointer[ds.Config] // configuration (atomic pointer for lock-free access)
	ClientAddr string                    // client address
	pollerOnce sync.Once                 // ensures poller is started only once
	AppInfo    ds.AppInfo                // combined application information
}

func MakeController(config ds.Config) (*ControllerImpl, error) {
	c := &ControllerImpl{}

	// Initialize AppInfo
	c.AppInfo.AppRunId = uuid.New().String()

	if config.AppName == "" {
		config.AppName = c.determineAppName()
	}
	c.AppInfo.AppName = config.AppName

	if config.ModuleName == "" {
		config.ModuleName = c.determineModuleName()
	}
	c.AppInfo.ModuleName = config.ModuleName

	c.configPtr.Store(&config)

	// Initialize the rest of AppInfo
	c.AppInfo.StartTime = time.Now().UnixMilli()
	c.AppInfo.Args = utilfn.CopyStrArr(os.Args)
	c.AppInfo.Executable, _ = os.Executable()
	c.AppInfo.Env = utilfn.CopyStrArr(os.Environ())
	c.AppInfo.Pid = os.Getpid()
	user, err := user.Current()
	if err == nil {
		c.AppInfo.User = user.Username
	}
	hostname, err := os.Hostname()
	if err == nil {
		c.AppInfo.Hostname = hostname
	}

	go c.runConnPoller()
	return c, nil
}

// Connection management methods

func (c *ControllerImpl) Connect() bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	if global.OutrigForceDisabled.Load() {
		return false
	}
	if global.OutrigEnabled.Load() {
		return false
	}

	atomic.StoreInt64(&global.TransportErrors, 0)
	var conn net.Conn
	var err error
	cfg := c.configPtr.Load()
	if cfg == nil {
		return false
	}

	// Attempt domain socket if not disabled
	if cfg.DomainSocketPath != "-" {
		dsPath := utilfn.ExpandHomeDir(cfg.DomainSocketPath)
		if _, errStat := os.Stat(dsPath); errStat == nil {
			conn, err = net.DialTimeout("unix", dsPath, 2*time.Second)
			if err == nil {
				fmt.Println("Outrig connected via domain socket:", dsPath)
				c.conn.Store(&conn)
				c.ClientAddr = cfg.DomainSocketPath
				c.sendAppInfo()
				global.OutrigEnabled.Store(true)
				go c.onConnect()
				return true
			}
		}
	}

	// Fall back to TCP if not disabled
	if cfg.ServerAddr != "-" {
		conn, err = net.DialTimeout("tcp", cfg.ServerAddr, 2*time.Second)
		if err == nil {
			fmt.Println("Outrig connected via TCP:", cfg.ServerAddr)
			c.conn.Store(&conn)
			c.ClientAddr = cfg.ServerAddr
			c.sendAppInfo()
			global.OutrigEnabled.Store(true)
			go c.onConnect()
			return true
		}
	}

	return false
}

func (c *ControllerImpl) Disconnect() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	global.OutrigConnected.Store(false)
	global.OutrigEnabled.Store(false)

	connPtr := c.conn.Load()
	if connPtr == nil {
		return
	}

	conn := *connPtr
	c.conn.Store(nil)
	time.Sleep(100 * time.Millisecond)
	conn.Close()
}

func (c *ControllerImpl) Enable() {
	global.OutrigForceDisabled.Store(false)

	if global.OutrigConnected.Load() {
		global.OutrigEnabled.Store(true)
	}

	c.Connect()
}

func (c *ControllerImpl) Disable(disconnect bool) {
	global.OutrigForceDisabled.Store(true)
	global.OutrigEnabled.Store(false)

	if disconnect {
		c.Disconnect()
	}
}

// Configuration methods

func (c *ControllerImpl) GetConfig() *ds.Config {
	return c.configPtr.Load()
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
		atomic.AddInt64(&global.TransportErrors, 1) // this will force a disconnect later
		return false, nil
	}

	atomic.AddInt64(&global.TransportPacketsSent, 1)
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

func (c *ControllerImpl) onConnect() {
	global.OutrigConnected.Store(true)

	collector := logprocess.GetInstance()
	collector.OnFirstConnect()
}

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
		if atomic.LoadInt64(&global.TransportErrors) > 0 {
			c.Disconnect()
			return
		}
		return
	} else {
		c.Connect()
	}
}
