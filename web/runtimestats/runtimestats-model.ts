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
    pollingInterval: number = 5000; // 5 seconds by default
    pollingIntervalId: number | null = null;

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
    }

    // Clean up resources when component unmounts
    dispose() {
        this.stopPolling();
    }

    // Start polling for runtime stats
    startPolling() {
        if (this.pollingIntervalId !== null) {
            this.stopPolling();
        }

        // Initial fetch
        this.refresh();

        // Set up interval for polling
        this.pollingIntervalId = window.setInterval(() => {
            this.refresh();
        }, this.pollingInterval);
    }

    // Stop polling for runtime stats
    stopPolling() {
        if (this.pollingIntervalId !== null) {
            window.clearInterval(this.pollingIntervalId);
            this.pollingIntervalId = null;
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
}

export { RuntimeStatsModel };
