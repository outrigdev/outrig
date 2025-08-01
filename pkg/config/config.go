// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

const OutrigSDKVersion = "v0.9.0"

// Environment variables
const (
	DomainSocketEnvName       = "OUTRIG_DOMAINSOCKET"
	TcpAddrEnvName            = "OUTRIG_TCPADDR"
	DisabledEnvName           = "OUTRIG_DISABLED"
	NoTelemetryEnvName        = "OUTRIG_NOTELEMETRY"
	DevConfigEnvName          = "OUTRIG_DEVCONFIG"
	DisableDockerProbeEnvName = "OUTRIG_DISABLEDOCKERPROBE"
	ExternalLogCaptureEnvName = "OUTRIG_EXTERNALLOGCAPTURE"
	AppRunIdEnvName           = "OUTRIG_APPRUNID"
	ConfigFileEnvName         = "OUTRIG_CONFIGFILE"
	ConfigJsonEnvName         = "OUTRIG_CONFIGJSON"
	OutrigPathEnvName         = "OUTRIG_OUTRIGBINPATH"
	AppNameEnvName            = "OUTRIG_APPNAME"
	RunSDKReplacePathEnvName  = "OUTRIG_RUN_SDKREPLACEPATH"
	FromRunModeEnvName        = "OUTRIG_FROMRUNMODE"
)

// Home directory paths
const (
	OutrigHome              = "~/.config/outrig"
	DevOutrigHome           = "~/.config/outrig-dev"
	DefaultDomainSocketName = "/outrig.sock"
)

// Default ports for the server (should match serverbase)
const (
	ProdWebServerPort = 5005
	DevWebServerPort  = 6005
)

var useDevConfig atomic.Bool

var (
	appRunIdOnce sync.Once
	appRunId     string
)

func init() {
	isDev := os.Getenv(DevConfigEnvName) != ""
	useDevConfig.Store(isDev)
}

type Config struct {
	Quiet bool `json:"quiet"` // If true, suppresses init, connect, and disconnect messages

	// AppName is the name of the application
	AppName string `json:"appname"`

	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string `json:"domainsocketpath"`

	// TcpAddr is the TCP address to connect to the Outrig server.  If "" => use default.
	// If "-" => disable TCP connection. Domain socket will be tried first (except on Windows where domain sockets are not supported).)
	TcpAddr string `json:"tcpaddr"`

	// By default the SDK will probe host.docker.internal:5005 to see if the Outrig monitor is running on the host machine
	// We do an initial DNS lookup at startup and only try this host/port if the DNS lookup succeeds.
	// Setting this to true will disable the initial probe.
	DisableDockerProbe bool `json:"disabledockerprobe"`

	// ModuleName is the name of the Go module. If not specified, it will be determined
	// from the go.mod file.
	ModuleName string `json:"modulename"`

	// If true, try to synchronously connect to the server on Init
	ConnectOnInit bool `json:"connectoninit"`

	// Collector configurations
	Collectors CollectorConfig `json:"collectors"`

	// RunMode configuration
	RunMode RunModeConfig `json:"runmode,omitempty"`

	// Exec options
	Exec ExecConfig `json:"exec,omitempty"`
}

type LogProcessorConfig struct {
	// Enabled indicates whether the log processor is enabled
	Enabled    bool `json:"enabled"`
	WrapStdout bool `json:"wrapstdout"`
	WrapStderr bool `json:"wrapstderr"`
	// OutrigPath is the full path to the outrig executable (including the executable name)
	// If empty, the system will look for "outrig" in the PATH
	OutrigPath string `json:"outrigpath"`
	// AdditionalArgs are additional arguments to pass to the outrig command
	// These are inserted before the "capturelogs" argument
	AdditionalArgs []string `json:"additionalargs"`
}

type WatchConfig struct {
	// Enabled indicates whether the watch collector is enabled
	Enabled bool `json:"enabled"`
}

type GoRoutineConfig struct {
	// Enabled indicates whether the goroutine collector is enabled
	Enabled bool `json:"enabled"`
}

type RuntimeStatsConfig struct {
	// Enabled indicates whether the runtime stats collector is enabled
	Enabled bool `json:"enabled"`
}

type CollectorConfig struct {
	Logs         LogProcessorConfig `json:"logs"`
	RuntimeStats RuntimeStatsConfig `json:"runtimestats"`
	Watch        WatchConfig        `json:"watch"`
	Goroutine    GoRoutineConfig    `json:"goroutine"`

	Plugins map[string]any `json:"-"`
}

type RunModeConfig struct {
	// SDKReplacePath specifies an absolute path to replace the outrig SDK import.
	// This must be an absolute path to a local outrig SDK directory.
	SDKReplacePath string `json:"sdkreplacepath,omitempty"`

	// TransformPkgs specifies a list of additional package patterns to transform
	TransformPkgs []string `json:"transformpkgs,omitempty"`
}

type ExecConfig struct {
	// Entry specifies the Go package or .go files to run (relative to config file location).
	// Examples: ".", "./cmd/myapp", "main.go", "cmd/myapp/main.go"
	// Must specify either Entry OR RawCmd, not both.
	Entry string `json:"entry,omitempty"`

	// BuildFlags are Go build flags to pass to the go run command.
	// Examples: ["-race", "-tags=debug", "-ldflags=-X main.version=1.0"]
	BuildFlags []string `json:"buildflags,omitempty"`

	// Args are command-line arguments to pass to the Go program after it's built.
	Args []string `json:"args,omitempty"`

	// Env specifies additional environment variables to set when running the program.
	Env map[string]string `json:"env,omitempty"`

	// Cwd specifies the working directory for the program (relative to config file location).
	// If not specified, defaults to the directory containing the config file.
	Cwd string `json:"cwd,omitempty"`

	// RawCmd specifies a raw shell command to execute instead of running Go code.
	// This runs through the shell, so $() and `` expansions will work.
	// Must specify either Entry OR RawCmd, not both.
	RawCmd string `json:"rawcmd,omitempty"`

	// RawCmdShell specifies which shell to use for RawCmd execution.
	// Defaults to $SHELL environment variable.
	RawCmdShell string `json:"rawcmdshell,omitempty"`
}

// getDefaultConfig returns a default configuration with the specified dev mode
func getDefaultConfig(isDev bool) *Config {
	return &Config{
		DomainSocketPath: GetDomainSocketNameForClient(),
		TcpAddr:          GetTcpAddrForClient(),
		ModuleName:       "",
		ConnectOnInit:    true,
		Collectors: CollectorConfig{
			Logs: LogProcessorConfig{
				Enabled:    true,
				WrapStdout: true,
				WrapStderr: true,
			},
			Watch: WatchConfig{
				Enabled: true,
			},
			Goroutine: GoRoutineConfig{
				Enabled: true,
			},
			RuntimeStats: RuntimeStatsConfig{
				Enabled: true,
			},
		},
	}
}

func GetAppRunId() string {
	appRunIdOnce.Do(func() {
		appRunId = os.Getenv(AppRunIdEnvName)
		if appRunId == "" {
			appRunId = uuid.New().String()
		} else {
			// Validate and normalize the UUID format
			if parsedUuid, err := uuid.Parse(appRunId); err != nil {
				appRunId = uuid.New().String()
			} else {
				appRunId = parsedUuid.String()
			}
		}
	})
	return appRunId
}

func GetExternalAppRunId() string {
	extAppRunId := os.Getenv(AppRunIdEnvName)
	if extAppRunId != GetAppRunId() {
		return ""
	}
	return extAppRunId
}

func UseDevConfig() bool {
	return useDevConfig.Load()
}

func SetUseDevConfig(dev bool) {
	useDevConfig.Store(dev)
}

// DefaultConfig returns the default configuration for normal usage
func DefaultConfig() *Config {
	if UseDevConfig() {
		return getDefaultConfig(true)
	}
	return getDefaultConfig(false)
}

// DefaultConfigForOutrigDevelopment returns a configuration specifically for Outrig internal development
// This is only used for internal Outrig development and not intended for general SDK users
func DefaultConfigForOutrigDevelopment() *Config {
	return getDefaultConfig(true)
}

func GetTcpAddrForClient() string {
	return "127.0.0.1:" + strconv.Itoa(GetMonitorPort())
}

func GetMonitorPort() int {
	if UseDevConfig() {
		return DevWebServerPort
	}
	return ProdWebServerPort
}

// GetOutrigHomeForClient returns the appropriate home directory based on client config
func GetOutrigHomeForClient() string {
	if UseDevConfig() {
		return DevOutrigHome
	}
	return OutrigHome
}

// GetDomainSocketNameForClient returns the full domain socket path for client
func GetDomainSocketNameForClient() string {
	return GetOutrigHomeForClient() + DefaultDomainSocketName
}

// UnmarshalJSON implements custom unmarshaling for Config with defaults
func (c *Config) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = *defaultConfig

	// Then unmarshal user values
	type alias Config
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for LogProcessorConfig with defaults
func (c *LogProcessorConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Collectors.Logs

	// Then unmarshal user values
	type alias LogProcessorConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for WatchConfig with defaults
func (c *WatchConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Collectors.Watch

	// Then unmarshal user values
	type alias WatchConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for GoRoutineConfig with defaults
func (c *GoRoutineConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Collectors.Goroutine

	// Then unmarshal user values
	type alias GoRoutineConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for RuntimeStatsConfig with defaults
func (c *RuntimeStatsConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Collectors.RuntimeStats

	// Then unmarshal user values
	type alias RuntimeStatsConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for RunModeConfig with defaults
func (c *RunModeConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.RunMode

	// Then unmarshal user values
	type alias RunModeConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for ExecConfig with defaults
func (c *ExecConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Exec

	// Then unmarshal user values
	type alias ExecConfig
	return json.Unmarshal(data, (*alias)(c))
}

// UnmarshalJSON implements custom unmarshaling for CollectorConfig with defaults
func (c *CollectorConfig) UnmarshalJSON(data []byte) error {
	// Set defaults first
	defaultConfig := getDefaultConfig(UseDevConfig())
	*c = defaultConfig.Collectors

	// First unmarshal into a generic map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle known fields
	if logs, ok := raw["logs"]; ok {
		if err := json.Unmarshal(logs, &c.Logs); err != nil {
			return err
		}
		delete(raw, "logs")
	}
	if runtimestats, ok := raw["runtimestats"]; ok {
		if err := json.Unmarshal(runtimestats, &c.RuntimeStats); err != nil {
			return err
		}
		delete(raw, "runtimestats")
	}
	if watch, ok := raw["watch"]; ok {
		if err := json.Unmarshal(watch, &c.Watch); err != nil {
			return err
		}
		delete(raw, "watch")
	}
	if goroutine, ok := raw["goroutine"]; ok {
		if err := json.Unmarshal(goroutine, &c.Goroutine); err != nil {
			return err
		}
		delete(raw, "goroutine")
	}

	// Everything else goes into Plugins as RawMessage
	c.Plugins = make(map[string]any)
	for k, v := range raw {
		c.Plugins[k] = v // v is json.RawMessage
	}

	return nil
}

// ValidateExecConfig validates the ExecConfig for consistency
func (e *ExecConfig) ValidateExecConfig() error {
	hasEntry := e.Entry != ""
	hasRawCmd := e.RawCmd != ""
	hasBuildFlags := len(e.BuildFlags) > 0
	hasArgs := len(e.Args) > 0

	// Must have either Entry OR RawCmd, not both
	if !hasEntry && !hasRawCmd {
		return fmt.Errorf("ExecConfig must have either 'entry' or 'rawcmd' specified")
	}

	if hasEntry && hasRawCmd {
		return fmt.Errorf("ExecConfig cannot have both 'entry' and 'rawcmd' specified")
	}

	// If you have buildflags or args, you MUST have an Entry (they're not compatible with rawcmd)
	if (hasBuildFlags || hasArgs) && !hasEntry {
		return fmt.Errorf("'buildflags' and 'args' are not compatible with 'rawcmd'")
	}

	return nil
}
