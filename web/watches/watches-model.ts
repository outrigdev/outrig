import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

class WatchesModel {
    widgetId: string;
    appRunId: string;
    appRunWatches: PrimitiveAtom<Watch[]> = atom<Watch[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    autoRefresh: PrimitiveAtom<boolean> = atom(true); // Default to on
    autoRefreshIntervalId: number | null = null;

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;

        // Start auto-refresh interval since default is on
        this.startAutoRefreshInterval();
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
            this.quietRefresh();
        }, 1000); // Refresh every second
    }

    // Stop the auto-refresh interval
    stopAutoRefreshInterval() {
        if (this.autoRefreshIntervalId !== null) {
            window.clearInterval(this.autoRefreshIntervalId);
            this.autoRefreshIntervalId = null;
        }
    }

    // Clean up resources when component unmounts
    dispose() {
        this.stopAutoRefreshInterval();
    }

    // Filtered watches based on search term
    filteredWatches: Atom<Watch[]> = atom((get): Watch[] => {
        const search = get(this.searchTerm);
        const watches = get(this.appRunWatches);

        // First sort by watch name
        const sortedWatches = [...watches].sort((a, b) => a.name.localeCompare(b.name));

        // Apply search filter if there's a search term
        if (!search) {
            return sortedWatches;
        }

        return sortedWatches.filter(
            (watch) =>
                watch.name.toLowerCase().includes(search.toLowerCase()) ||
                watch.type.toLowerCase().includes(search.toLowerCase()) ||
                (watch.value && watch.value.toLowerCase().includes(search.toLowerCase()))
        );
    });

    async fetchAppRunWatches() {
        try {
            const result = await RpcApi.GetAppRunWatchesCommand(DefaultRpcClient, { apprunid: this.appRunId });
            return result.watches;
        } catch (error) {
            console.error(`Failed to load watches for app run ${this.appRunId}:`, error);
            return [];
        }
    }

    // Load watches with a minimum time to show the refreshing state
    async loadAppRunWatches(minTime: number = 0) {
        const startTime = new Date().getTime();

        try {
            const watches = await this.fetchAppRunWatches();

            // If minTime is specified, ensure we wait at least that long
            if (minTime > 0) {
                const curTime = new Date().getTime();
                if (curTime - startTime < minTime) {
                    await new Promise((r) => setTimeout(r, minTime - (curTime - startTime)));
                }
            }

            getDefaultStore().set(this.appRunWatches, watches);
        } catch (error) {
            console.error(`Failed to load watches for app run ${this.appRunId}:`, error);
        }
    }

    // Refresh watches with a minimum time to show the refreshing state
    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Set refreshing state to true
        store.set(this.isRefreshing, true);

        // Clear watches immediately
        store.set(this.appRunWatches, []);

        try {
            // Load new watches with a minimum time of 500ms to show the refreshing state
            await this.loadAppRunWatches(500);
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }

    // Quiet refresh for auto-refresh - doesn't set isRefreshing or clear watches
    async quietRefresh() {
        // Get the app run info to check its status
        const store = getDefaultStore();
        const appRunInfoAtom = AppModel.getAppRunInfoAtom(this.appRunId);
        const appRunInfo = store.get(appRunInfoAtom);

        // If app run is not connected (status is not "running"), don't refresh
        if (!appRunInfo || appRunInfo.status !== "running") {
            return;
        }

        try {
            const watches = await this.fetchAppRunWatches();
            getDefaultStore().set(this.appRunWatches, watches);
        } catch (error) {
            console.error(`Failed to auto-refresh watches for app run ${this.appRunId}:`, error);
        }
    }
}

export { WatchesModel };
