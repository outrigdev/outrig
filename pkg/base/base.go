package base

// Connection mode constants
const (
	ConnectionModeCrashOutput = "crashoutput"
	ConnectionModePacket      = "packet"
)

// Home directory paths
const OutrigHome = "~/.outrig"
const DevOutrigHome = "~/.outrig-dev"

// Domain socket name (just the filename part)
const DefaultDomainSocketName = "/outrig.sock"

// Default production ports
const ProdTCPPort = 5005
const ProdTCPAddr = "http://localhost:5005"

// Development ports
const DevTCPPort = 6005
const DevTCPAddr = "http://localhost:6005"

// Default constants for backward compatibility
const DefaultTCPPort = ProdTCPPort
const DefaultTCPAddr = ProdTCPAddr

// Client-specific functions that use the client's Dev flag

// GetOutrigHomeForClient returns the appropriate home directory based on client config
func GetOutrigHomeForClient(isDev bool) string {
	if isDev {
		return DevOutrigHome
	}
	return OutrigHome
}

// GetDomainSocketNameForClient returns the full domain socket path for client
func GetDomainSocketNameForClient(isDev bool) string {
	return GetOutrigHomeForClient(isDev) + DefaultDomainSocketName
}

// GetTCPAddrForClient returns the appropriate TCP address based on client config
func GetTCPAddrForClient(isDev bool) string {
	if isDev {
		return DevTCPAddr
	}
	return ProdTCPAddr
}
