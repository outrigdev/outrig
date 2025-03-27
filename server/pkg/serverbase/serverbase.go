package serverbase

import (
	"os"
	"path/filepath"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// OutrigVersion is the current version of Outrig
// This gets set from main-server.go during initialization
var OutrigVersion = "v0.0.0"

// OutrigBuildTime is the build timestamp of Outrig
// This gets set from main-server.go during initialization
var OutrigBuildTime = ""

const OutrigLockFile = "outrig.lock"
const OutrigDataDir = "data"
const OutrigDevEnvName = "OUTRIG_DEV"

// Default production ports for server
const ProdWebServerPort = 5005
const ProdWebSocketPort = 5006

// Development ports for server
const DevWebServerPort = 6005
const DevWebSocketPort = 6006

type FDLock interface {
	Close() error
}

// IsDev returns true if the server is running in development mode
func IsDev() bool {
	return os.Getenv(OutrigDevEnvName) == "1"
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

// GetOutrigDataDir returns the path to the data directory
func GetOutrigDataDir() string {
	return filepath.Join(GetOutrigHome(), OutrigDataDir)
}

func EnsureHomeDir() error {
	outrigHomeDir := utilfn.ExpandHomeDir(GetOutrigHome())
	return os.MkdirAll(outrigHomeDir, 0755)
}

func EnsureDataDir() error {
	dataDir := utilfn.ExpandHomeDir(GetOutrigDataDir())
	return os.MkdirAll(dataDir, 0755)
}
