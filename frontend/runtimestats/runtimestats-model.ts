// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { RpcApi } from "@/rpc/rpcclientapi";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

// Maximum number of runtime stats entries to keep (10 minutes of data at 1s intervals)
const MAX_RUNTIME_STATS_ENTRIES = 600;

// Create a type that combines the AppRunRuntimeStatsData with a single RuntimeStatData
// This is for backward compatibility with the existing UI
export type CombinedStatsData = {
    apprunid: string;
    appname: string;
    ts: number;
    cpuusage: number;
    goroutinecount: number;
    numactivegoroutines: number;
    numoutriggoroutines: number;
    gomaxprocs: number;
    numcpu: number;
    goos: string;
    goarch: string;
    goversion: string;
    pid: number;
    cwd: string;
    memstats: MemoryStatsInfo;
};

class RuntimeStatsModel {
    widgetId: string;
    appRunId: string;
    // Store all runtime stats
    allRuntimeStats: PrimitiveAtom<RuntimeStatData[]> = atom<RuntimeStatData[]>([]) as PrimitiveAtom<RuntimeStatData[]>;
    // Store the latest timestamp we've seen
    latestTimestamp: PrimitiveAtom<number> = atom<number>(0) as PrimitiveAtom<number>;
    // For backward compatibility with the UI, we'll keep a single stats object
    runtimeStats: PrimitiveAtom<CombinedStatsData | null> = atom<CombinedStatsData | null>(
        null
    ) as PrimitiveAtom<CombinedStatsData | null>;
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    autoRefresh: PrimitiveAtom<boolean> = atom(true); // Default to on
    autoRefreshIntervalId: number | null = null;
    autoRefreshInterval: number = 1000; // 1 second by default

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;

        // Initial refresh
        this.quietRefresh(true);

        // Start auto-refresh interval since default is on
        this.startAutoRefreshInterval();
    }

    // Clean up resources when component unmounts
    dispose() {
        this.stopAutoRefreshInterval();
    }

    // Toggle auto-refresh state
    toggleAutoRefresh() {
        const store = getDefaultStore();
        const currentState = store.get(this.autoRefresh);
        store.set(this.autoRefresh, !currentState);

        if (!currentState) {
            // If turning on, start the interval
            this.startAutoRefreshInterval();
        } else {
            // If turning off, clear the interval
            this.stopAutoRefreshInterval();
        }
    }

    // Start the auto-refresh interval
    startAutoRefreshInterval() {
        // Clear any existing interval first
        this.stopAutoRefreshInterval();

        // Set up new interval
        this.autoRefreshIntervalId = window.setInterval(() => {
            this.quietRefresh(false);
        }, this.autoRefreshInterval);
    }

    // Stop the auto-refresh interval
    stopAutoRefreshInterval() {
        if (this.autoRefreshIntervalId !== null) {
            window.clearInterval(this.autoRefreshIntervalId);
            this.autoRefreshIntervalId = null;
        }
    }

    // Fetch runtime stats from the backend
    async fetchRuntimeStats(): Promise<AppRunRuntimeStatsData | null> {
        try {
            const store = getDefaultStore();
            const sinceTs = store.get(this.latestTimestamp);

            // Call the RPC API to get runtime stats with the latest timestamp
            const result = await RpcApi.GetAppRunRuntimeStatsCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                since: sinceTs,
            });

            return result;
        } catch (error) {
            console.error(`Failed to load runtime stats for app run ${this.appRunId}:`, error);
            return null;
        }
    }

    // Update the latest timestamp based on the stats we received
    private updateLatestTimestamp(stats: RuntimeStatData[]) {
        if (stats.length === 0) return;

        const store = getDefaultStore();
        const currentLatestTs = store.get(this.latestTimestamp);

        // Find the maximum timestamp in the new stats
        const maxTs = Math.max(...stats.map((stat) => stat.ts));

        // Update the latest timestamp if the new max is greater
        if (maxTs > currentLatestTs) {
            store.set(this.latestTimestamp, maxTs);
        }
    }

    // Update the legacy runtime stats object for backward compatibility with the UI
    private updateLegacyRuntimeStats(result: AppRunRuntimeStatsData) {
        if (result.stats.length === 0) return;

        const store = getDefaultStore();

        // Use the most recent stat for the legacy object
        const latestStat = result.stats[result.stats.length - 1];

        // Create a legacy-format object from the latest stat
        const legacyStats: CombinedStatsData = {
            apprunid: result.apprunid,
            appname: result.appname,
            ts: latestStat.ts,
            cpuusage: latestStat.cpuusage,
            goroutinecount: latestStat.goroutinecount,
            numactivegoroutines: result.numactivegoroutines,
            numoutriggoroutines: result.numoutriggoroutines,
            gomaxprocs: latestStat.gomaxprocs,
            numcpu: latestStat.numcpu,
            goos: latestStat.goos,
            goarch: latestStat.goarch,
            goversion: latestStat.goversion,
            pid: latestStat.pid,
            cwd: latestStat.cwd,
            memstats: latestStat.memstats,
        };

        // Update the legacy stats atom
        store.set(this.runtimeStats, legacyStats);
    }

    // Refresh runtime stats with a minimum time to show the refreshing state
    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Set refreshing state to true
        store.set(this.isRefreshing, true);

        try {
            // Fetch new stats
            const result = await this.fetchRuntimeStats();

            if (result && result.stats.length > 0) {
                // Get current stats
                const currentStats = store.get(this.allRuntimeStats);

                // Append new stats
                let updatedStats = [...currentStats, ...result.stats];

                // Limit the array size to MAX_RUNTIME_STATS_ENTRIES
                if (updatedStats.length > MAX_RUNTIME_STATS_ENTRIES) {
                    updatedStats = updatedStats.slice(-MAX_RUNTIME_STATS_ENTRIES);
                }

                // Update all stats atom
                store.set(this.allRuntimeStats, updatedStats);

                // Update the latest timestamp
                this.updateLatestTimestamp(result.stats);

                // Update the legacy stats object for backward compatibility
                this.updateLegacyRuntimeStats(result);
            }
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }

    // Quiet refresh for auto-refresh - doesn't set isRefreshing or clear stats
    async quietRefresh(force: boolean) {
        // Get the app run info to check its status
        const store = getDefaultStore();
        const appRunInfoAtom = AppModel.getAppRunInfoAtom(this.appRunId);
        const appRunInfo = store.get(appRunInfoAtom);

        if (!appRunInfo) {
            return;
        }

        // If app run is not connected (status is not "running"), don't refresh
        if (!force && appRunInfo.status !== "running") {
            return;
        }

        try {
            // Fetch new stats
            const result = await this.fetchRuntimeStats();

            if (result && result.stats.length > 0) {
                // Get current stats
                const currentStats = store.get(this.allRuntimeStats);

                // Append new stats
                let updatedStats = [...currentStats, ...result.stats];

                // Limit the array size to MAX_RUNTIME_STATS_ENTRIES
                if (updatedStats.length > MAX_RUNTIME_STATS_ENTRIES) {
                    updatedStats = updatedStats.slice(-MAX_RUNTIME_STATS_ENTRIES);
                }

                // Update all stats atom
                store.set(this.allRuntimeStats, updatedStats);

                // Update the latest timestamp
                this.updateLatestTimestamp(result.stats);

                // Update the legacy stats object for backward compatibility
                this.updateLegacyRuntimeStats(result);
            }
        } catch (error) {
            console.error(`Failed to auto-refresh runtime stats for app run ${this.appRunId}:`, error);
        }
    }
}

export { RuntimeStatsModel };
