import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { RpcApi } from "@/rpc/rpcclientapi";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

class RuntimeStatsModel {
    widgetId: string;
    appRunId: string;
    runtimeStats: PrimitiveAtom<AppRunRuntimeStatsData | null> = atom<AppRunRuntimeStatsData | null>(
        null
    ) as PrimitiveAtom<AppRunRuntimeStatsData | null>;
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
            // Call the RPC API to get runtime stats
            const result = await RpcApi.GetAppRunRuntimeStatsCommand(DefaultRpcClient, { apprunid: this.appRunId });
            
            // Return the result directly since it's already in the correct format
            return result;
        } catch (error) {
            console.error(`Failed to load runtime stats for app run ${this.appRunId}:`, error);
            return null;
        }
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
            const stats = await this.fetchRuntimeStats();

            // Update the atom with the new stats
            store.set(this.runtimeStats, stats);
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
            const stats = await this.fetchRuntimeStats();
            if (stats) {
                getDefaultStore().set(this.runtimeStats, stats);
            }
        } catch (error) {
            console.error(`Failed to auto-refresh runtime stats for app run ${this.appRunId}:`, error);
        }
    }
}

export { RuntimeStatsModel };
