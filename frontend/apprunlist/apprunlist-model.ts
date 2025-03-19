import { DefaultRpcClient } from "@/init";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";
import { RpcApi } from "../rpc/rpcclientapi";
import { mergeArraysByKey } from "../util/util";
import { addWSReconnectHandler } from "../websocket/client";

class AppRunModel {
    // App runs data
    appRuns: PrimitiveAtom<AppRunInfo[]> = atom<AppRunInfo[]>([]);

    // Track the last time we fetched app run updates (in milliseconds)
    appRunsInfoLastUpdateTime: number = 0;

    // Flag to indicate we need a full refresh of app runs data after reconnection
    needsFullAppRunsRefresh: boolean = false;

    // Flag to track if this is the first load after page initialization
    isInitialLoad: boolean = true;

    appRunsTimeoutId: NodeJS.Timeout = null;

    constructor() {
        this.appRunsTimeoutId = setInterval(() => {
            // Catch errors in the interval to prevent it from stopping
            this.loadAppRuns().catch((error) => {
                console.error("Failed to load app runs in interval:", error);
            });
        }, 1000);

        // Register a WebSocket reconnect handler to force a full refresh when connection is reestablished
        addWSReconnectHandler(this.handleServerReconnect.bind(this));
    }

    // Handle server reconnection by forcing a full refresh of app runs
    handleServerReconnect() {
        console.log("[AppRunModel] WebSocket reconnected, will perform full refresh of app runs");
        this.needsFullAppRunsRefresh = true;

        // Trigger an immediate refresh but catch any errors to prevent unhandled rejections
        this.loadAppRuns().catch((error) => {
            console.error("[AppRunModel] Error refreshing app runs after reconnection:", error);
        });
    }

    async loadAppRuns() {
        console.log("[AppRunModel] Loading app runs, isInitialLoad:", this.isInitialLoad);
        
        // If we need a full refresh, reset the lastUpdateTime to 0
        if (this.needsFullAppRunsRefresh) {
            console.log("[AppRunModel] Performing full refresh of app runs after reconnection");
            this.appRunsInfoLastUpdateTime = 0;
        }

        // Get app runs with incremental updates (or full list if since=0)
        const result = await RpcApi.GetAppRunsCommand(DefaultRpcClient, { since: this.appRunsInfoLastUpdateTime });

        if (this.needsFullAppRunsRefresh) {
            // For a full refresh, completely replace the app runs list
            getDefaultStore().set(this.appRuns, result.appruns);
            this.needsFullAppRunsRefresh = false;
        } else {
            // For incremental updates, merge with existing app runs
            const currentAppRuns = getDefaultStore().get(this.appRuns);
            const updatedAppRuns = mergeArraysByKey(currentAppRuns, result.appruns, (run) => run.apprunid);
            getDefaultStore().set(this.appRuns, updatedAppRuns);
        }

        // Update the last update time to the maximum lastmodtime from all app runs
        // This is more robust than using the client's time (avoids clock skew issues)
        if (result.appruns.length > 0) {
            const maxLastModTime = Math.max(...result.appruns.map((run) => run.lastmodtime));
            // Only update if the new max time is greater than the current value
            if (maxLastModTime > this.appRunsInfoLastUpdateTime) {
                this.appRunsInfoLastUpdateTime = maxLastModTime;
            }
        }
        // If there are no app runs or no newer timestamps, keep the previous lastUpdateTime value

        // Handle auto-follow functionality, but skip toast on initial load
        this.handleAutoFollow(this.isInitialLoad);
        
        // After first load, set isInitialLoad to false
        if (this.isInitialLoad) {
            this.isInitialLoad = false;
        }
    }

    // Find the "best" app run (running with latest start time)
    findBestAppRun(): AppRunInfo {
        const appRuns = getDefaultStore().get(this.appRuns);
        if (!appRuns || appRuns.length === 0) {
            return null;
        }

        // Sort app runs: first by running status (running first), then by start time (newest first)
        const sortedAppRuns = [...appRuns].sort((a, b) => {
            // First sort by running status
            if (a.isrunning && !b.isrunning) return -1;
            if (!a.isrunning && b.isrunning) return 1;

            // Then sort by start time (newest first)
            return b.starttime - a.starttime;
        });

        // Return the first (best) app run
        return sortedAppRuns[0];
    }

    // Handle auto-follow logic
    handleAutoFollow(isInitialLoad: boolean) {
        console.log("[AppRunModel] Handling auto-follow, isInitialLoad:", isInitialLoad);
        const autoFollow = getDefaultStore().get(AppModel.autoFollow);
        if (!autoFollow) {
            return; // Auto-follow is disabled, do nothing
        }

        const currentAppRunId = getDefaultStore().get(AppModel.selectedAppRunId);
        const bestAppRun = this.findBestAppRun();

        // If there's no best app run and we have a current selection, clear it and go to app runs tab
        if (!bestAppRun && currentAppRunId) {
            console.log(`[AutoFollow] No app runs available, clearing selection`);
            AppModel.clearAppRunSelection();
            return;
        }

        // If there's no best app run or no current selection, do nothing
        if (!bestAppRun || !bestAppRun.apprunid) {
            return;
        }
        
        // Check if we're already on the best app run
        if (bestAppRun.apprunid === currentAppRunId) {
            // We're already on the best app run, no need to switch or show toast
            return;
        }

        // If our current app run is not the best, switch to the best but stay on current tab
        console.log(`[AutoFollow] Switching from ${currentAppRunId || 'none'} to ${bestAppRun.apprunid}`);
        
        // First switch to the new app run
        AppModel.selectAppRunKeepTab(bestAppRun.apprunid, true); // true indicates this is an auto-follow selection
        
        // Only show a toast notification if we're actually switching to a different app run
        // AND this is not the initial page load
        if (!isInitialLoad) {
            const appName = bestAppRun.appname || "Unknown";
            const shortId = bestAppRun.apprunid.substring(0, 8); // First 8 chars of the ID
            
            // Format the start time information
            const startTimeInfo = this.formatStartTimeInfo(bestAppRun.starttime);
            
            AppModel.showToast(
                "App Run Switched", 
                `Auto-switched to App Run ${appName} (${shortId}) ${startTimeInfo}`,
                3000 // 3 seconds timeout
            );
        }
    }
    
    // Format the start time information based on how recent it is
    private formatStartTimeInfo(startTimeMs: number): string {
        const now = Date.now();
        const diffSeconds = (now - startTimeMs) / 1000;
        
        // If started within the last 5 seconds
        if (diffSeconds < 5) {
            return "started just now";
        }
        
        // Format the date/time
        const startDate = new Date(startTimeMs);
        
        // If it's today, just show the time
        const today = new Date();
        if (startDate.toDateString() === today.toDateString()) {
            return `started at ${startDate.toLocaleTimeString()}`;
        }
        
        // Otherwise show the full date and time
        return `started at ${startDate.toLocaleString()}`;
    }
}

export { AppRunModel };
