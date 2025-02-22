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
	"github.com/outrigdev/outrig/pkg/logprocess"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second

var ConfigPtr atomic.Pointer[ds.Config]
var CoreLock *sync.Mutex = &sync.Mutex{}
var PollerRunning int32 = 0
var LogsWrapInitialized bool = false

func SetConfig(cfg *ds.Config) {
	CoreLock.Lock()
	defer CoreLock.Unlock()
	ConfigPtr.Store(cfg)
}

func Disconnect() {
	CoreLock.Lock()
	defer CoreLock.Unlock()
	atomic.StoreInt32(&global.OutrigConnected, 0)
	client := global.ClientPtr.Load()
	atomic.StoreInt32(&global.OutrigEnabled, 0)
	if client == nil {
		return
	}
	global.ClientPtr.Store(nil)
	time.Sleep(100 * time.Millisecond)
	client.Conn.Close()
}

func RunConnPoller() {
	ok := atomic.CompareAndSwapInt32(&PollerRunning, 0, 1)
	if !ok {
		return
	}
	defer atomic.StoreInt32(&PollerRunning, 0)
	for {
		PollConn()
		time.Sleep(ConnPollTime)
	}
}

func PollConn() {
	enabled := atomic.LoadInt32(&global.OutrigEnabled)
	if enabled != 0 {
		// check for errors
		if atomic.LoadInt64(&global.TransportErrors) > 0 {
			Disconnect()
			return
		}
		return
	} else {
		TryConnect()
	}
}

func onConnect() {
	atomic.StoreInt32(&global.OutrigConnected, 1)
	logprocess.OnFirstConnect()
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
	atomic.StoreInt64(&global.TransportErrors, 0)
	var c net.Conn
	var err error
	cfg := ConfigPtr.Load()
	// Attempt domain socket if not disabled
	if cfg.DomainSocketPath != "-" {
		dsPath := utilfn.ExpandHomeDir(cfg.DomainSocketPath)
		if _, errStat := os.Stat(dsPath); errStat == nil {
			c, err = net.DialTimeout("unix", dsPath, 2*time.Second)
			if err == nil {
				fmt.Println("Outrig connected via domain socket:", dsPath)
				global.ClientPtr.Store(&ds.ClientType{
					Conn:       c,
					ClientAddr: cfg.DomainSocketPath,
				})
				atomic.StoreInt32(&global.OutrigEnabled, 1)
				go onConnect()
				return
			}
		}
	}

	// Fall back to TCP if not disabled
	if cfg.ServerAddr != "-" {
		c, err = net.DialTimeout("tcp", cfg.ServerAddr, 2*time.Second)
		if err == nil {
			fmt.Println("Outrig connected via TCP:", cfg.ServerAddr)
			global.ClientPtr.Store(&ds.ClientType{
				Conn:       c,
				ClientAddr: cfg.ServerAddr,
			})
			atomic.StoreInt32(&global.OutrigEnabled, 1)
			go onConnect()
			return
		}
	}
}
