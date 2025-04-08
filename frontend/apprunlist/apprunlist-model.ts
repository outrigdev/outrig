// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { DefaultRpcClient } from "@/init";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { selectAtom } from "jotai/utils";
import { AppModel } from "../appmodel";
import { RpcApi } from "../rpc/rpcclientapi";
import { mergeArraysByKey } from "../util/util";
import { addWSReconnectHandler } from "../websocket/client";

class AppRunModel {
    appRuns: PrimitiveAtom<AppRunInfo[]> = atom<AppRunInfo[]>([]);
    appRunCount = selectAtom(this.appRuns, (appRuns) => appRuns.length);
    
    // Track the last time we fetched app run updates (in milliseconds)
    appRunsInfoLastUpdateTime: number = 0;
    needsFullAppRunsRefresh: boolean = false;
    isInitialLoad: boolean = true;
    appRunsTimeoutId: NodeJS.Timeout = null;
    previousRunningStates = new Map<string, boolean>();

    constructor() {
        this.appRunsTimeoutId = setInterval(() => {
            this.loadAppRuns().catch((error) => {
                console.error("Failed to load app runs in interval:", error);
            });
        }, 1000);

        addWSReconnectHandler(this.handleServerReconnect.bind(this));
    }

    handleServerReconnect() {
        console.log("[AppRunModel] WebSocket reconnected, will perform full refresh of app runs");
        this.needsFullAppRunsRefresh = true;

        this.loadAppRuns().catch((error) => {
            console.error("[AppRunModel] Error refreshing app runs after reconnection:", error);
        });
    }

    async loadAppRuns() {
        if (this.needsFullAppRunsRefresh) {
            console.log("[AppRunModel] Performing full refresh of app runs after reconnection");
            this.appRunsInfoLastUpdateTime = 0;
        }

        const result = await RpcApi.GetAppRunsCommand(DefaultRpcClient, { since: this.appRunsInfoLastUpdateTime });
        const currentAppRuns = getDefaultStore().get(this.appRuns);

        let updatedAppRuns: AppRunInfo[];
        if (this.needsFullAppRunsRefresh) {
            updatedAppRuns = result.appruns;
            getDefaultStore().set(this.appRuns, updatedAppRuns);
            this.needsFullAppRunsRefresh = false;
        } else {
            updatedAppRuns = mergeArraysByKey(currentAppRuns, result.appruns, (run) => run.apprunid);
            getDefaultStore().set(this.appRuns, updatedAppRuns);
        }

        if (!this.isInitialLoad) {
            this.checkForRunningStatusChanges(currentAppRuns, updatedAppRuns);
        } else {
            updatedAppRuns.forEach((run) => {
                this.previousRunningStates.set(run.apprunid, run.isrunning);
            });
        }

        if (result.appruns.length > 0) {
            const maxLastModTime = Math.max(...result.appruns.map((run) => run.lastmodtime));
            if (maxLastModTime > this.appRunsInfoLastUpdateTime) {
                this.appRunsInfoLastUpdateTime = maxLastModTime;
            }
        }

        this.handleAutoFollow(this.isInitialLoad);

        if (this.isInitialLoad) {
            this.isInitialLoad = false;
        }
    }

    checkForRunningStatusChanges(previousAppRuns: AppRunInfo[], currentAppRuns: AppRunInfo[]) {
        const selectedAppRunId = getDefaultStore().get(AppModel.selectedAppRunId);
        if (!selectedAppRunId) return;

        const currentAppRun = currentAppRuns.find((run) => run.apprunid === selectedAppRunId);
        if (!currentAppRun) return;

        const previousRunningState = this.previousRunningStates.get(selectedAppRunId);

        if (previousRunningState !== undefined && previousRunningState !== currentAppRun.isrunning) {
            if (currentAppRun.isrunning) {
                AppModel.showToast(
                    "App Run Re-Connected",
                    `App run ${currentAppRun.appname || "Unknown"} (${selectedAppRunId.substring(0, 4)}) has reconnected`,
                    5000
                );
            } else {
                const statusMessage = currentAppRun.status === "done" ? "completed" : "disconnected";
                AppModel.showToast(
                    "App Run Disconnected",
                    `App run ${currentAppRun.appname || "Unknown"} (${selectedAppRunId.substring(0, 4)}) has ${statusMessage}`,
                    5000
                );
            }
        }

        currentAppRuns.forEach((run) => {
            this.previousRunningStates.set(run.apprunid, run.isrunning);
        });
    }

    // Find the "best" app run (running with latest start time)
    // If appName is provided, only consider app runs with that app name
    findBestAppRun(appName?: string): AppRunInfo {
        const appRuns = getDefaultStore().get(this.appRuns);
        if (!appRuns || appRuns.length === 0) {
            return null;
        }

        const filteredAppRuns = appName 
            ? appRuns.filter(run => run.appname === appName)
            : appRuns;
        
        if (filteredAppRuns.length === 0) {
            return null;
        }

        const sortedAppRuns = [...filteredAppRuns].sort((a, b) => {
            if (a.isrunning && !b.isrunning) return -1;
            if (!a.isrunning && b.isrunning) return 1;
            return b.starttime - a.starttime;
        });

        return sortedAppRuns[0];
    }

    handleAutoFollow(isInitialLoad: boolean) {
        const autoFollow = getDefaultStore().get(AppModel.autoFollow);
        if (!autoFollow) {
            return; // Auto-follow is disabled, do nothing
        }

        const currentAppRunId = getDefaultStore().get(AppModel.selectedAppRunId);
        if (!currentAppRunId) {
            // If we're on the homepage (no app run selected), don't auto-follow
            return;
        }

        const appRuns = getDefaultStore().get(this.appRuns);
        const currentAppRun = appRuns.find(run => run.apprunid === currentAppRunId);
        if (!currentAppRun) {
            // Current app run not found in the list, do nothing
            return;
        }

        const bestAppRun = this.findBestAppRun(currentAppRun.appname);

        if (!bestAppRun || !bestAppRun.apprunid) {
            return;
        }

        if (bestAppRun.apprunid === currentAppRunId) {
            // We're already on the best app run, no need to switch or show toast
            return;
        }

        console.log(`[AutoFollow] Switching from ${currentAppRunId} to ${bestAppRun.apprunid} (same app: ${currentAppRun.appname})`);

        const currentTab = getDefaultStore().get(AppModel.selectedTab);
        AppModel.selectAppRunKeepTab(bestAppRun.apprunid, true);

        if (!isInitialLoad) {
            const appName = bestAppRun.appname || "Unknown";
            const shortId = bestAppRun.apprunid.substring(0, 4);
            const startTimeInfo = this.formatStartTimeInfo(bestAppRun.starttime);

            AppModel.showToast(
                "App Run Switched",
                `Auto-switched to app run ${appName} (${shortId}) ${startTimeInfo}`,
                3000
            );
        }
    }

    formatStartTimeInfo(startTimeMs: number): string {
        const now = Date.now();
        const diffSeconds = (now - startTimeMs) / 1000;

        if (diffSeconds < 5) {
            return "started just now";
        }

        const startDate = new Date(startTimeMs);
        const today = new Date();
        if (startDate.toDateString() === today.toDateString()) {
            return `started at ${startDate.toLocaleTimeString()}`;
        }

        return `started at ${startDate.toLocaleString()}`;
    }
}

export { AppRunModel };
