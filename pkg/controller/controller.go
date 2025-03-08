package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/logprocess"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second

type ControllerImpl struct {
	Lock          sync.Mutex // lock for this struct
	Conn          net.Conn   // connection to server
	ClientAddr    string     // client address
	pollerRunning int32      // atomic flag for connection poller
}

func NewController() *ControllerImpl {
	return &ControllerImpl{}
}

func (c *ControllerImpl) SetAsGlobal() {
	global.GlobalController = c
}

// Connection management methods

func (c *ControllerImpl) Connect() bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	if atomic.LoadInt32(&global.OutrigForceDisabled) != 0 {
		return false
	}
	if atomic.LoadInt32(&global.OutrigEnabled) != 0 {
		return false
	}

	atomic.StoreInt64(&global.TransportErrors, 0)
	var conn net.Conn
	var err error
	cfg := global.ConfigPtr.Load()
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
				c.Conn = conn
				c.ClientAddr = cfg.DomainSocketPath
				global.ClientPtr.Store(&ds.ClientType{
					Conn:       conn,
					ClientAddr: cfg.DomainSocketPath,
				})
				atomic.StoreInt32(&global.OutrigEnabled, 1)
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
			c.Conn = conn
			c.ClientAddr = cfg.ServerAddr
			global.ClientPtr.Store(&ds.ClientType{
				Conn:       conn,
				ClientAddr: cfg.ServerAddr,
			})
			atomic.StoreInt32(&global.OutrigEnabled, 1)
			go c.onConnect()
			return true
		}
	}

	return false
}

func (c *ControllerImpl) Disconnect() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	atomic.StoreInt32(&global.OutrigConnected, 0)
	atomic.StoreInt32(&global.OutrigEnabled, 0)

	if c.Conn == nil {
		return
	}

	conn := c.Conn
	c.Conn = nil
	global.ClientPtr.Store(nil)
	time.Sleep(100 * time.Millisecond)
	conn.Close()
}

func (c *ControllerImpl) IsConnected() bool {
	return atomic.LoadInt32(&global.OutrigConnected) != 0
}

func (c *ControllerImpl) IsEnabled() bool {
	return atomic.LoadInt32(&global.OutrigEnabled) != 0
}

func (c *ControllerImpl) Enable() {
	atomic.StoreInt32(&global.OutrigForceDisabled, 0)

	if atomic.LoadInt32(&global.OutrigConnected) != 0 {
		atomic.StoreInt32(&global.OutrigEnabled, 1)
	}

	c.Connect()
}

func (c *ControllerImpl) Disable(disconnect bool) {
	atomic.StoreInt32(&global.OutrigForceDisabled, 1)
	atomic.StoreInt32(&global.OutrigEnabled, 0)

	if disconnect {
		c.Disconnect()
	}
}

// Configuration methods

func (c *ControllerImpl) GetConfig() *ds.Config {
	return global.ConfigPtr.Load()
}

func (c *ControllerImpl) SetConfig(cfg *ds.Config) {
	global.ConfigPtr.Store(cfg)
}

// Transport methods

func (c *ControllerImpl) SendPacket(pk *ds.PacketType) (bool, error) {
	if atomic.LoadInt32(&global.OutrigEnabled) == 0 {
		return false, nil
	}

	client := global.ClientPtr.Load()
	if client == nil {
		return false, nil
	}

	barr, err := json.Marshal(pk)
	if err != nil {
		return false, err
	}

	barr = append(barr, '\n')
	_, err = client.Conn.Write(barr)
	if err != nil {
		atomic.AddInt64(&global.TransportErrors, 1) // this will force a disconnect later
		return false, nil
	}

	atomic.AddInt64(&global.TransportPacketsSent, 1)
	return true, nil
}

func (c *ControllerImpl) GetTransportStats() (int64, int64) {
	return atomic.LoadInt64(&global.TransportErrors), atomic.LoadInt64(&global.TransportPacketsSent)
}

// Initialization methods

func (c *ControllerImpl) Init(cfgParam *ds.Config) error {
	if cfgParam != nil {
		c.SetConfig(cfgParam)
	}

	go c.runConnPoller()
	return nil
}

func (c *ControllerImpl) Shutdown() {
	// TODO: wait for last log lines to be sent
	// TODO: send shutdown log lines
	c.Disconnect()
}

// Private methods

func (c *ControllerImpl) onConnect() {
	atomic.StoreInt32(&global.OutrigConnected, 1)
	logprocess.OnFirstConnect()
}

func (c *ControllerImpl) runConnPoller() {
	ok := atomic.CompareAndSwapInt32(&c.pollerRunning, 0, 1)
	if !ok {
		return
	}
	defer atomic.StoreInt32(&c.pollerRunning, 0)

	for {
		c.pollConn()
		time.Sleep(ConnPollTime)
	}
}

func (c *ControllerImpl) pollConn() {
	enabled := atomic.LoadInt32(&global.OutrigEnabled)
	if enabled != 0 {
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
