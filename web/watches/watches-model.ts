import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

class WatchesModel {
    widgetId: string;
    appRunId: string;
    appRunWatches: PrimitiveAtom<Watch[]> = atom<Watch[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
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
}

export { WatchesModel };
