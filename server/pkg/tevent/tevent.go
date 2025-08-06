// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

// extractMajorMinorVersion extracts just the major.minor version (e.g., "v0.2") from a full version string
// It handles formats like "v0.2.0-alpha", "v0.2.0", etc.
func extractMajorMinorVersion(fullVersion string) string {
	// Parse the version using semver
	v, err := semver.NewVersion(fullVersion)
	if err != nil {
		// If parsing fails, return empty string
		return ""
	}

	// Always format as vMajor.Minor with the v prefix
	return fmt.Sprintf("v%d.%d", v.Major(), v.Minor())
}

var Disabled atomic.Bool
var IsTrayApp atomic.Bool

var ValidEventNames = map[string]bool{
	"server:install":  true,
	"server:startup":  true,
	"server:shutdown": true,
	"server:activity": true,
	"server:panic":    true,

	"apprun:connected":    true,
	"apprun:disconnected": true,
	"apprun:activity":     true,

	"frontend:tab":              true,
	"frontend:selectapprun":     true,
	"frontend:click":            true,
	"frontend:activity":         true,
	"frontend:error":            true,
	"frontend:homepage":         true,
	"frontend:search:logs":      true,
	"frontend:search:goroutine": true,
	"frontend:search:watch":     true,
}

var osLangOnce = &sync.Once{}
var osLang string
var releaseRegex = regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
var osReleaseOnce = &sync.Once{}
var osRelease string

// AppRunStats contains statistics about an app run session
type AppRunStats struct {
	LogLines    int    `json:"apprun:loglines,omitempty"`
	GoRoutines  int    `json:"apprun:goroutines,omitempty"`
	Watches     int    `json:"apprun:watches,omitempty"`
	Collections int    `json:"apprun:collections,omitempty"`
	SDKVersion  string `json:"apprun:sdkversion,omitempty"`
	ConnTimeMs  int64  `json:"apprun:conntimems,omitempty"`
	AppRunCount int    `json:"apprun:count,omitempty"`
}

// Sub subtracts another AppRunStats from this one and returns the result
func (s AppRunStats) Sub(other AppRunStats) AppRunStats {
	return AppRunStats{
		LogLines:    s.LogLines - other.LogLines,
		GoRoutines:  s.GoRoutines - other.GoRoutines,
		Watches:     s.Watches - other.Watches,
		Collections: s.Collections - other.Collections,
		SDKVersion:  s.SDKVersion,
		ConnTimeMs:  s.ConnTimeMs - other.ConnTimeMs,
		AppRunCount: s.AppRunCount - other.AppRunCount,
	}
}

// Add adds another AppRunStats to this one and returns the result
func (s AppRunStats) Add(other AppRunStats) AppRunStats {
	return AppRunStats{
		LogLines:    s.LogLines + other.LogLines,
		GoRoutines:  s.GoRoutines + other.GoRoutines,
		Watches:     s.Watches + other.Watches,
		Collections: s.Collections + other.Collections,
		SDKVersion:  s.SDKVersion,
		ConnTimeMs:  s.ConnTimeMs + other.ConnTimeMs,
		AppRunCount: s.AppRunCount + other.AppRunCount,
	}
}

type TEvent struct {
	Uuid    string      `json:"uuid,omitempty"`
	Ts      int64       `json:"ts,omitempty"`
	TsLocal string      `json:"tslocal,omitempty"` // iso8601 format (wall clock converted to PT)
	Event   string      `json:"event"`
	Props   TEventProps `json:"props"` // Don't scan directly to map
}

type TEventUserProps struct {
	ServerArch           string `json:"server:arch,omitempty"`
	ServerVersion        string `json:"server:version,omitempty"`
	ServerFullVersion    string `json:"server:fullversion,omitempty"`
	ServerInitialVersion string `json:"server:initial_version,omitempty"`
	ServerBuildTime      string `json:"server:buildtime,omitempty"`
	ServerBuildCommit    string `json:"server:buildcommit,omitempty"`
	ServerOSRelease      string `json:"server:osrelease,omitempty"`
	ServerIsDev          bool   `json:"server:isdev,omitempty"`
	ServerLang           string `json:"server:lang,omitempty"`
	LocCountryCode       string `json:"loc:countrycode,omitempty"`
	LocRegionCode        string `json:"loc:regioncode,omitempty"`
}

type TEventProps struct {
	TEventUserProps `tstype:"-"` // generally don't need to set these since they will be automatically copied over

	PanicType string `json:"debug:panictype,omitempty"`

	FrontendTab            string   `json:"frontend:tab,omitempty"`
	FrontendSearchFeatures []string `json:"frontend:logsearchfeatures,omitempty"`
	FrontendSearchLatency  int      `json:"frontend:searchlatency,omitempty"`
	FrontendSearchItems    int      `json:"frontend:searchitems,omitempty"`
	FrontendClickType      string   `json:"frontend:clicktype,omitempty"`

	ServerNumAppRuns int  `json:"server:numappruns,omitempty"`
	ServerNumApps    int  `json:"server:numapps,omitempty"`
	ServerTrayApp    bool `json:"server:trayapp,omitempty"`

	// counts for app run activity
	AppRunLogLines       int    `json:"apprun:loglines,omitempty"`
	AppRunGoRoutines     int    `json:"apprun:goroutines,omitempty"`
	AppRunWatches        int    `json:"apprun:watches,omitempty"`
	AppRunCollections    int    `json:"apprun:collections,omitempty"`
	AppRunSDKVersion     string `json:"apprun:sdkversion,omitempty"`
	AppRunSDKFullVersion string `json:"apprun:sdkfullversion,omitempty"`
	AppRunGoVersion      string `json:"apprun:goversion,omitempty"`
	AppRunConnTimeMs     int64  `json:"apprun:conntimems,omitempty"`
	AppRunCount          int    `json:"apprun:count,omitempty"`
	AppRunDemo           bool   `json:"apprun:demo,omitempty"`
	AppRunRunMode        bool   `json:"apprun:runmode,omitempty"`

	UserSet     *TEventUserProps `json:"$set,omitempty"`
	UserSetOnce *TEventUserProps `json:"$set_once,omitempty"`
}

// ApplyAppRunStats applies the fields from an AppRunStats struct to this TEventProps
func (p *TEventProps) ApplyAppRunStats(stats AppRunStats) {
	p.AppRunLogLines = stats.LogLines
	p.AppRunGoRoutines = stats.GoRoutines
	p.AppRunWatches = stats.Watches
	p.AppRunCollections = stats.Collections
	p.AppRunSDKFullVersion = stats.SDKVersion
	p.AppRunSDKVersion = extractMajorMinorVersion(stats.SDKVersion)
	p.AppRunConnTimeMs = stats.ConnTimeMs
	p.AppRunCount = stats.AppRunCount
}

func MakeTEvent(event string, props TEventProps) *TEvent {
	now := time.Now()
	// TsLocal gets set in EnsureTimestamps()
	return &TEvent{
		Uuid:  uuid.New().String(),
		Ts:    now.UnixMilli(),
		Event: event,
		Props: props,
	}
}

func MakeUntypedTEvent(event string, propsMap map[string]any) (*TEvent, error) {
	if event == "" {
		return nil, fmt.Errorf("event name must be non-empty")
	}
	var props TEventProps
	err := utilfn.ReUnmarshal(&props, propsMap)
	if err != nil {
		return nil, fmt.Errorf("error re-marshalling TEvent props: %w", err)
	}
	return MakeTEvent(event, props), nil
}

func (t *TEvent) EnsureTimestamps() {
	if t.Ts == 0 {
		t.Ts = time.Now().UnixMilli()
	}
	gtime := time.UnixMilli(t.Ts)
	t.TsLocal = utilfn.ConvertToWallClockPT(gtime).Format(time.RFC3339)
}

func (t *TEvent) UserSetProps() *TEventUserProps {
	if t.Props.UserSet == nil {
		t.Props.UserSet = &TEventUserProps{}
	}
	return t.Props.UserSet
}

func (t *TEvent) UserSetOnceProps() *TEventUserProps {
	if t.Props.UserSetOnce == nil {
		t.Props.UserSetOnce = &TEventUserProps{}
	}
	return t.Props.UserSetOnce
}

var eventNameRe = regexp.MustCompile(`^[a-zA-Z0-9.:_/-]+$`)

// validates a tevent that was just created (not for validating out of the DB, or an uploaded TEvent)
// checks that TS is pretty current (or unset)
func (te *TEvent) Validate(current bool) error {
	if te == nil {
		return fmt.Errorf("TEvent cannot be nil")
	}
	if te.Event == "" {
		return fmt.Errorf("TEvent.Event cannot be empty")
	}
	if !eventNameRe.MatchString(te.Event) {
		return fmt.Errorf("TEvent.Event invalid: %q", te.Event)
	}
	if !ValidEventNames[te.Event] {
		return fmt.Errorf("TEvent.Event not valid: %q", te.Event)
	}
	if te.Uuid == "" {
		return fmt.Errorf("TEvent.Uuid cannot be empty")
	}
	_, err := uuid.Parse(te.Uuid)
	if err != nil {
		return fmt.Errorf("TEvent.Uuid invalid: %v", err)
	}
	if current {
		if te.Ts != 0 {
			now := time.Now().UnixMilli()
			if te.Ts > now+60000 || te.Ts < now-60000 {
				return fmt.Errorf("TEvent.Ts is not current: %d", te.Ts)
			}
		}
	} else {
		if te.Ts == 0 {
			return fmt.Errorf("TEvent.Ts must be set")
		}
		if te.TsLocal == "" {
			return fmt.Errorf("TEvent.TsLocal must be set")
		}
		t, err := time.Parse(time.RFC3339, te.TsLocal)
		if err != nil {
			return fmt.Errorf("TEvent.TsLocal parse error: %v", err)
		}
		now := time.Now()
		if t.Before(now.Add(-30*24*time.Hour)) || t.After(now.Add(2*24*time.Hour)) {
			return fmt.Errorf("tslocal out of valid range")
		}
	}
	barr, err := json.Marshal(te.Props)
	if err != nil {
		return fmt.Errorf("TEvent.Props JSON error: %v", err)
	}
	if len(barr) > 20000 {
		return fmt.Errorf("TEvent.Props too large: %d", len(barr))
	}
	return nil
}

func ClientArch() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func unameKernelRelease() string {
	ctx, cancelFn := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFn()
	out, err := exec.CommandContext(ctx, "uname", "-r").CombinedOutput()
	if err != nil {
		log.Printf("error executing uname -r: %v\n", err)
		return "-"
	}
	releaseStr := strings.TrimSpace(string(out))
	m := releaseRegex.FindStringSubmatch(releaseStr)
	if len(m) < 2 {
		log.Printf("invalid uname -r output: [%s]\n", releaseStr)
		return "-"
	}
	return m[1]
}

func UnameKernelRelease() string {
	osReleaseOnce.Do(func() {
		osRelease = unameKernelRelease()
	})
	return osRelease
}

func determineLang() string {
	defaultLang := "en_US"
	ctx, cancelFn := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFn()
	if runtime.GOOS == "darwin" {
		out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLocale").CombinedOutput()
		if err != nil {
			log.Printf("error executing 'defaults read -g AppleLocale', will use default 'en_US.UTF-8': %v\n", err)
			return defaultLang
		}
		strOut := string(out)
		truncOut := strings.Split(strOut, "@")[0]
		preferredLang := strings.TrimSpace(truncOut)
		return preferredLang
	} else {
		return os.Getenv("LANG")
	}
}

func OsLang() string {
	osLangOnce.Do(func() {
		osLang = determineLang()
	})
	return osLang
}

// createCommonUserProps creates a TEventUserProps with common properties set
func createCommonUserProps() *TEventUserProps {
	return &TEventUserProps{
		ServerArch:        ClientArch(),
		ServerFullVersion: serverbase.OutrigServerVersion,
		ServerVersion:     extractMajorMinorVersion(serverbase.OutrigServerVersion),
		ServerBuildTime:   serverbase.OutrigBuildTime,
		ServerBuildCommit: serverbase.OutrigCommit,
		ServerOSRelease:   UnameKernelRelease(),
		ServerIsDev:       serverbase.IsDev(),
		ServerLang:        OsLang(),
	}
}

// SetTrayApp sets whether the server was started from the tray app
func SetTrayApp(isTrayApp bool) {
	IsTrayApp.Store(isTrayApp)
}

// SendInstallEvent sends an "outrig:install" telemetry event
func SendInstallEvent() {
	if Disabled.Load() {
		return
	}
	props := TEventProps{}
	props.UserSet = createCommonUserProps()
	props.ServerTrayApp = IsTrayApp.Load()
	event := MakeTEvent("server:install", props)
	event.UserSetOnceProps().ServerInitialVersion = serverbase.OutrigServerVersion
	WriteTEvent(*event)
}

// SendStartupEvent sends a "server:startup" telemetry event
func SendStartupEvent() {
	if Disabled.Load() {
		return
	}
	props := TEventProps{}
	props.UserSet = createCommonUserProps()
	props.ServerTrayApp = IsTrayApp.Load()
	event := MakeTEvent("server:startup", props)
	WriteTEvent(*event)
}

// SendShutdownEvent sends a "server:shutdown" telemetry event
func SendShutdownEvent() {
	if Disabled.Load() {
		return
	}
	event := MakeTEvent("server:shutdown", TEventProps{})
	WriteTEvent(*event)
}

// SendAppRunConnectedEvent sends an "apprun:connected" telemetry event
func SendAppRunConnectedEvent(sdkVersion string, goVersion string, appName string, runMode bool) {
	if Disabled.Load() {
		return
	}
	props := TEventProps{
		AppRunSDKFullVersion: sdkVersion,
		AppRunSDKVersion:     extractMajorMinorVersion(sdkVersion),
		AppRunGoVersion:      goVersion,
		AppRunDemo:           isDemo(appName),
		AppRunRunMode:        runMode,
	}
	event := MakeTEvent("apprun:connected", props)
	WriteTEvent(*event)
}

// isDemo returns true if the app name indicates this is a demo application
func isDemo(appName string) bool {
	return appName == "OutrigAcres"
}

// SendAppRunDisconnectedEvent sends an "apprun:disconnected" telemetry event
func SendAppRunDisconnectedEvent(stats AppRunStats) {
	if Disabled.Load() {
		return
	}
	props := TEventProps{}
	props.ApplyAppRunStats(stats)
	event := MakeTEvent("apprun:disconnected", props)
	WriteTEvent(*event)
}

// SendServerActivityEvent sends a "server:activity" telemetry event with stats
// aggregated across all app runs
func SendServerActivityEvent(stats AppRunStats, numActiveAppRuns int) {
	if Disabled.Load() {
		return
	}
	props := TEventProps{}
	props.ApplyAppRunStats(stats)
	props.ServerNumAppRuns = numActiveAppRuns
	props.ServerTrayApp = IsTrayApp.Load()
	event := MakeTEvent("server:activity", props)
	WriteTEvent(*event)
}
