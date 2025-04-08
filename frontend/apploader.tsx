// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { useAtomValue } from "jotai";
import { useEffect, useState } from "react";
import { App } from "./app";
import { AppModel } from "./appmodel";
import { GlobalWS } from "./init";

/**
 * AppLoader component that waits for the websocket connection to be established
 * and app runs to be loaded before rendering the main App component.
 * 
 * On the happy path, we wait for both the websocket connection and app runs to load.
 * If the websocket fails, we render the App component immediately without waiting for app runs.
 * Once we've resolved once, we always render the App component even if the connection state changes.
 */
function AppLoader() {
    const connectionState = useAtomValue(GlobalWS.connectionState);
    const [hasResolvedOnce, setHasResolvedOnce] = useState(false);
    const [appRunsLoaded, setAppRunsLoaded] = useState(false);
    const [isLoadingAppRuns, setIsLoadingAppRuns] = useState(false);
    
    useEffect(() => {
        // If the connection state is "failed", mark it as resolved immediately
        if (connectionState === "failed") {
            setHasResolvedOnce(true);
        }
        
        // If the connection is established and we haven't started loading app runs yet,
        // start loading them
        if (connectionState === "connected" && !isLoadingAppRuns && !appRunsLoaded) {
            setIsLoadingAppRuns(true);
            
            // Load app runs
            AppModel.loadAppRuns()
                .then(() => {
                    setAppRunsLoaded(true);
                    setHasResolvedOnce(true);
                })
                .catch((error) => {
                    console.error("Failed to load app runs:", error);
                    // Even if loading app runs fails, we should still render the app
                    setHasResolvedOnce(true);
                });
        }
    }, [connectionState, isLoadingAppRuns, appRunsLoaded]);

    // If we haven't resolved at least once and either:
    // 1. We're still connecting, or
    // 2. We're connected but still loading app runs
    // Then return null (show loading state)
    if (!hasResolvedOnce && (connectionState === "connecting" || (connectionState === "connected" && !appRunsLoaded))) {
        return null;
    }

    // Otherwise, render the App component
    return <App />;
}

export { AppLoader };
