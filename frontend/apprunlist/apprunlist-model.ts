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

        // Handle auto-follow functionality
        this.handleAutoFollow();
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
    handleAutoFollow() {
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

        // If there's no best app run or no current selection, or they match, do nothing
        if (!bestAppRun || !bestAppRun.apprunid || bestAppRun.apprunid === currentAppRunId) {
            return;
        }

        // If our current app run is not the best, switch to the best but stay on current tab
        console.log(`[AutoFollow] Switching from ${currentAppRunId || "none"} to ${bestAppRun.apprunid}`);
        AppModel.selectAppRunKeepTab(bestAppRun.apprunid);
    }
}

export { AppRunModel };
