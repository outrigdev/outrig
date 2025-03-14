import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

// This will be replaced with actual types when we hook up the backend
interface RuntimeStats {
    // Placeholder for runtime statistics
    timestamp: number;
    memoryUsage: number;
    cpuUsage: number;
    goroutineCount: number;
    // Add more stats as needed
}

class RuntimeStatsModel {
    widgetId: string;
    appRunId: string;
    runtimeStats: PrimitiveAtom<RuntimeStats | null> = atom<RuntimeStats | null>(null) as PrimitiveAtom<RuntimeStats | null>;
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
    async fetchRuntimeStats(): Promise<RuntimeStats | null> {
        try {
            // This will be replaced with actual API call when we hook up the backend
            // For now, return mock data
            return {
                timestamp: Date.now(),
                memoryUsage: Math.random() * 1000,
                cpuUsage: Math.random() * 100,
                goroutineCount: Math.floor(Math.random() * 1000),
            };

            // When we hook up the backend, it will look something like:
            // const result = await RpcApi.GetRuntimeStatsCommand(DefaultRpcClient, { apprunid: this.appRunId });
            // return result.stats;
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
