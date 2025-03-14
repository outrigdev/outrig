import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";

class WatchesModel {
    widgetId: string;
    appRunId: string;
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
    }

    // Filtered watches based on search term
    filteredWatches: Atom<any[]> = atom((get): any[] => {
        const search = get(this.searchTerm);
        
        // For now, return an empty array since we don't have actual watches yet
        return [];
    });

    // Refresh watches with a minimum time to show the refreshing state
    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Set refreshing state to true
        store.set(this.isRefreshing, true);

        try {
            // In the future, we'll load watches here
            await new Promise(resolve => setTimeout(resolve, 500));
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }
}

export { WatchesModel };
