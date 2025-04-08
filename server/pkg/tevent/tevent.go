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

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

var Disabled atomic.Bool

var ValidEventNames = map[string]bool{
	"server:install":            true,
	"server:startup":            true,
	"server:shutdown":           true,
	"server:activity":           true,
	"server:panic":              true,
	"apprun:connected":          true,
	"apprun:disconnected":       true,
	"apprun:activity":           true,
	"frontend:tab":              true,
	"frontend:selectapprun":     true,
	"frontend:click":            true,
	"frontend:activity":         true,
	"frontend:error":            true,
	"frontend:search:logs":      true,
	"frontend:search:goroutine": true,
	"frontend:search:watch":     true,
}

var osLangOnce = &sync.Once{}
var osLang string
var releaseRegex = regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
var osReleaseOnce = &sync.Once{}
var osRelease string

type TEvent struct {
	Uuid    string      `json:"uuid,omitempty"`
	Ts      int64       `json:"ts,omitempty"`
	TsLocal string      `json:"tslocal,omitempty"` // iso8601 format (wall clock converted to PT)
	Event   string      `json:"event"`
	Props   TEventProps `json:"props"` // Don't scan directly to map
}

type TEventUserProps struct {
	ClientArch           string `json:"client:arch,omitempty"`
	ClientVersion        string `json:"client:version,omitempty"`
	ClientInitialVersion string `json:"client:initial_version,omitempty"`
	ClientBuildTime      string `json:"client:buildtime,omitempty"`
	ClientOSRelease      string `json:"client:osrelease,omitempty"`
	ClientIsDev          bool   `json:"client:isdev,omitempty"`
}

type TEventProps struct {
	TEventUserProps `tstype:"-"` // generally don't need to set these since they will be automatically copied over

	PanicType string `json:"debug:panictype,omitempty"`
	ClickType string `json:"debug:clicktype,omitempty"`

	FrontendTab            string   `json:"frontend:tab,omitempty"`
	FrontendSearchFeatures []string `json:"frontend:logsearchfeatures,omitempty"`
	FrontendSearchLatency  int      `json:"frontend:searchlatency,omitempty"`
	FrontendSearchItems    int      `json:"frontend:searchitems,omitempty"`

	ServerNumAppRuns int `json:"server:numappruns,omitempty"`
	ServerNumApps    int `json:"server:numapps,omitempty"`

	// counts for app run activity
	AppRunLogLines    int `json:"apprun:loglines,omitempty"`
	AppRunGoRoutines  int `json:"apprun:goroutines,omitempty"`
	AppRunConnTime    int `json:"apprun:conntime,omitempty"`
	AppRunWatches     int `json:"apprun:watches,omitempty"`
	AppRunCollections int `json:"apprun:collections,omitempty"`

	UserSet     *TEventUserProps `json:"$set,omitempty"`
	UserSetOnce *TEventUserProps `json:"$set_once,omitempty"`
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
	if m == nil || len(m) < 2 {
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
