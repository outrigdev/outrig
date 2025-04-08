// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package browsertabs

import (
	"fmt"
	"log"
	"sync"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

// Constants
const (
	BrowserTabsRouteId = "browsertabs"
)

// BrowserTabInfo stores information about a browser tab
type BrowserTabInfo struct {
	AppRunId   string `json:"apprunid"`
	Url        string `json:"url"`
	Focused    bool   `json:"focused"`
	AutoFollow bool   `json:"autofollow"`
}

// Global map to store route ID to browser tab info mapping
var (
	browserTabsMutex sync.Mutex
	browserTabs      = make(map[string]BrowserTabInfo)
)

// RPC client for browser tabs
var browserTabsRpcClient *rpc.RpcClient

// Initialize sets up the browser tabs tracking system
func Initialize() {
	// Create a new RPC client for browser tabs
	browserTabsRpcClient = rpc.MakeRpcClient(nil, nil, nil, BrowserTabsRouteId)

	// Register the client with the router
	rpc.DefaultRouter.RegisterRoute(BrowserTabsRouteId, browserTabsRpcClient, true)

	// Subscribe to route down events
	rpc.Broker.Subscribe(BrowserTabsRouteId, rpctypes.SubscriptionRequest{
		Event:     rpctypes.Event_RouteDown,
		AllScopes: true,
	})

	// Register an event handler for route down events
	browserTabsRpcClient.EventListener.On(rpctypes.Event_RouteDown, func(event *rpctypes.EventType) {
		if len(event.Scopes) > 0 {
			routeId := event.Scopes[0]
			log.Printf("[browsertabs] Route down event for %s", routeId)
			RemoveBrowserTab(routeId)
		}
	})

	outrig.WatchSync("browsertabs", &browserTabsMutex, &browserTabs)

	log.Printf("[browsertabs] Subscribed to route down events")
}

// UpdateBrowserTab updates or adds a browser tab in the tracking map
func UpdateBrowserTab(routeId string, data rpctypes.BrowserTabUrlData) {
	browserTabsMutex.Lock()
	defer browserTabsMutex.Unlock()

	browserTabs[routeId] = BrowserTabInfo{
		Url:        data.Url,
		AppRunId:   data.AppRunId,
		Focused:    data.Focused,
		AutoFollow: data.AutoFollow,
	}
}

// GetBrowserTabs returns a copy of the current browser tabs map
func GetBrowserTabs() map[string]BrowserTabInfo {
	browserTabsMutex.Lock()
	defer browserTabsMutex.Unlock()

	// Create a copy of the map to avoid concurrent access issues
	result := make(map[string]BrowserTabInfo, len(browserTabs))
	for routeId, info := range browserTabs {
		result[routeId] = info
	}

	return result
}

// GetBrowserTabsForAppName returns a list of browser tabs for a specific app name
func GetBrowserTabsForAppName(appName string) []string {
	browserTabsMutex.Lock()
	defer browserTabsMutex.Unlock()

	// Find all tabs with the given app name
	var result []string
	for routeId, info := range browserTabs {
		if info.AppRunId != "" {
			// Get the app run peer to check the app name
			peer := apppeer.GetAppRunPeer(info.AppRunId, false)
			if peer != nil && peer.AppInfo != nil && peer.AppInfo.AppName == appName {
				result = append(result, routeId)
			}
		}
	}

	return result
}

// RemoveBrowserTab removes a browser tab from the tracking map
func RemoveBrowserTab(routeId string) {
	browserTabsMutex.Lock()
	defer browserTabsMutex.Unlock()

	delete(browserTabs, routeId)
}

// HandleRouteDown handles route down events to clean up browser tabs
func HandleRouteDown(routeId string) {
	if routeId != "" {
		log.Printf("[browsertabs] Removing browser tab for route %s", routeId)
		RemoveBrowserTab(routeId)
	}
}

// UpdateBrowserTabUrl updates the URL for a browser tab
func UpdateBrowserTabUrl(routeId string, data rpctypes.BrowserTabUrlData) error {
	if routeId == "" {
		return fmt.Errorf("no route ID provided")
	}
	UpdateBrowserTab(routeId, data)
	return nil
}
