// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { DefaultRpcClient } from "./init";
import { RpcApi } from "./rpc/rpcclientapi";

// Track the last sent tab event to avoid duplicates
let lastTabEvent: { tab: string; timestamp: number } | null = null;

// Minimum time between tab events in milliseconds (to prevent duplicates from React dev mode and HMR)
const MIN_EVENT_INTERVAL = 1000;

/**
 * Send a frontend:tab event when a tab is selected
 * This function includes debouncing to prevent duplicate events from React dev mode and HMR
 *
 * @param tabName The name of the selected tab (logs, goroutines, watches, or runtimestats)
 */
export function sendTabEvent(tabName: string): void {
    // Skip if RPC client is not initialized
    if (!DefaultRpcClient) {
        return;
    }

    const now = Date.now();

    // Check if this is a duplicate event (same tab within MIN_EVENT_INTERVAL)
    if (lastTabEvent && lastTabEvent.tab === tabName && now - lastTabEvent.timestamp < MIN_EVENT_INTERVAL) {
        // Skip duplicate event
        return;
    }

    // Update last event tracking
    lastTabEvent = {
        tab: tabName,
        timestamp: now,
    };

    const teventData: TEventFeData = {
        event: "frontend:tab",
        props: {
            "frontend:tab": tabName,
        },
    };
    // Send the event to the backend
    RpcApi.SendTEventFeCommand(DefaultRpcClient, teventData).catch((err: Error) => {
        console.error("Failed to send tab event:", err);
    });
}

/**
 * Send a frontend:click event with a specific click type
 *
 * @param clickType The type of click event (e.g., "addwatch")
 */
export function sendClickEvent(clickType: string): void {
    // Skip if RPC client is not initialized
    if (!DefaultRpcClient) {
        return;
    }

    const teventData: TEventFeData = {
        event: "frontend:click",
        props: {
            "frontend:clicktype": clickType,
        },
    };
    // Send the event to the backend
    RpcApi.SendTEventFeCommand(DefaultRpcClient, teventData).catch((err: Error) => {
        console.error("Failed to send click event:", err);
    });
}

/**
 * Send a frontend:homepage event when navigating to the homepage
 */
export function sendHomepageEvent(): void {
    // Skip if RPC client is not initialized
    if (!DefaultRpcClient) {
        return;
    }

    const teventData: TEventFeData = {
        event: "frontend:homepage",
        props: {},
    };
    // Send the event to the backend
    RpcApi.SendTEventFeCommand(DefaultRpcClient, teventData).catch((err: Error) => {
        console.error("Failed to send homepage event:", err);
    });
}
