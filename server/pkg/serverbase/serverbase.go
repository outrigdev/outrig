package serverbase

import (
	"os"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const OutrigLockFile = "outrig.lock"

// Default production ports for server
const ProdWebServerPort = 5005
const ProdWebSocketPort = 5006

// Development ports for server
const DevWebServerPort = 6005
const DevWebSocketPort = 6006

type FDLock interface {
	Close() error
}

// IsDev is a wrapper around base.IsDev for convenience
func IsDev() bool {
	return base.IsDev()
}

// GetOutrigHome returns the appropriate home directory based on mode
func GetOutrigHome() string {
	if IsDev() {
		return base.DevOutrigHome
	}
	return base.OutrigHome
}

// GetDomainSocketName returns the full domain socket path
func GetDomainSocketName() string {
	return GetOutrigHome() + base.DefaultDomainSocketName
}

// GetWebServerPort returns the appropriate web server port based on mode
func GetWebServerPort() int {
	if IsDev() {
		return DevWebServerPort
	}
	return ProdWebServerPort
}

// GetWebSocketPort returns the appropriate websocket port based on mode
func GetWebSocketPort() int {
	if IsDev() {
		return DevWebSocketPort
	}
	return ProdWebSocketPort
}

func EnsureHomeDir() error {
	outrigHomeDir := utilfn.ExpandHomeDir(GetOutrigHome())
	return os.MkdirAll(outrigHomeDir, 0755)
}
