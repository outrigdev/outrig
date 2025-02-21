package core

import (
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

// ClientType represents our active connection client
type ClientType struct {
	Conn       net.Conn
	ClientAddr string
}

// clientHolder is an atomic container for *ClientType
var ClientPtr atomic.Pointer[ClientType]
var ConfigPtr atomic.Pointer[ds.Config]
var CoreLock *sync.Mutex = &sync.Mutex{}

func SetConfig(cfg *ds.Config) {
	CoreLock.Lock()
	defer CoreLock.Unlock()
	ConfigPtr.Store(cfg)
}

func Disconnect() {
	CoreLock.Lock()
	defer CoreLock.Unlock()
	client := ClientPtr.Load()
	if client == nil {
		atomic.StoreInt32(&global.OutrigEnabled, 0)
		return
	}
	atomic.StoreInt32(&global.OutrigEnabled, 0)
	ClientPtr.Store(nil)
	client.Conn.Close()
}

func TryConnect() {
	CoreLock.Lock()
	defer CoreLock.Unlock()
	if atomic.LoadInt32(&global.OutrigForceDisabled) != 0 {
		return
	}
	if atomic.LoadInt32(&global.OutrigEnabled) != 0 {
		return
	}
	var c net.Conn
	var err error
	cfg := ConfigPtr.Load()
	// Attempt domain socket if not disabled
	if cfg.DomainSocketPath != "-" {
		if _, errStat := os.Stat(cfg.DomainSocketPath); errStat == nil {
			c, err = net.DialTimeout("unix", cfg.DomainSocketPath, 2*time.Second)
			if err == nil {
				fmt.Println("Connected via domain socket:", cfg.DomainSocketPath)
				ClientPtr.Store(&ClientType{
					Conn:       c,
					ClientAddr: cfg.DomainSocketPath,
				})
				atomic.StoreInt32(&global.OutrigEnabled, 1)
				return
			}
		}
	}

	// Fall back to TCP if not disabled
	if cfg.ServerAddr != "-" {
		c, err = net.DialTimeout("tcp", cfg.ServerAddr, 2*time.Second)
		if err == nil {
			fmt.Println("Connected via TCP:", cfg.ServerAddr)
			ClientPtr.Store(&ClientType{
				Conn:       c,
				ClientAddr: cfg.ServerAddr,
			})
			atomic.StoreInt32(&global.OutrigEnabled, 1)
			return
		}
	}
}
