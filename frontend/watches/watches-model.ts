// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { SearchStore } from "@/store/searchstore";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

// Type for search result info
export type SearchResultInfo = {
    searchedCount: number;
    totalCount: number;
    errorSpans?: SearchErrorSpan[];
};

class WatchesModel {
    widgetId: string;
    appRunId: string;
    appRunWatches: PrimitiveAtom<CombinedWatchSample[]> = atom<CombinedWatchSample[]>([]);
    matchedWatchIds: PrimitiveAtom<number[]> = atom<number[]>([]);
    pinnedWatchNums: PrimitiveAtom<Set<number>> = atom<Set<number>>(new Set<number>());
    searchResultInfo: PrimitiveAtom<SearchResultInfo> = atom<SearchResultInfo>({
        searchedCount: 0,
        totalCount: 0,
        errorSpans: [],
    });
    searchTerm: PrimitiveAtom<string>;
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    autoRefresh: PrimitiveAtom<boolean> = atom(true); // Default to on
    autoRefreshIntervalId: number | null = null;
    contentRef: React.RefObject<HTMLDivElement> = null;
    currentSearchId: string = "";
    pinnedAtomCache: Map<number, Atom<boolean>> = new Map();

    // Total count of watches (derived from appRunWatches)
    totalCount: Atom<number> = atom((get) => {
        const watches = get(this.appRunWatches);
        return watches.length;
    });

    // Actual result count (derived from matchedWatchIds)
    resultCount: Atom<number> = atom((get) => {
        const matchedIds = get(this.matchedWatchIds);
        return matchedIds.length;
    });

    // Filtered count of watches (derived from filteredWatches)
    filteredCount: Atom<number> = atom((get) => {
        const filtered = get(this.filteredWatches);
        return filtered.length;
    });

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;

        // Get app name from AppModel using the appRunId
        const appRunInfoAtom = AppModel.getAppRunInfoAtom(appRunId);
        const appRunInfo = getDefaultStore().get(appRunInfoAtom);
        const appName = appRunInfo?.appname || "unknown";

        // Get search term atom from SearchStore
        this.searchTerm = SearchStore.getSearchTermAtom(appName, appRunId, "watches");

        // Initial refresh
        this.quietRefresh(true);

        // Start auto-refresh interval since default is on
        this.startAutoRefreshInterval();
    }

    // Set the content div reference for scrolling
    setContentRef(ref: React.RefObject<HTMLDivElement>) {
        this.contentRef = ref;
    }

    // Page up in the content view
    pageUp() {
        if (!this.contentRef?.current) return;

        this.contentRef.current.scrollBy({
            top: -500,
            behavior: "auto",
        });
    }

    // Page down in the content view
    pageDown() {
        if (!this.contentRef?.current) return;

        this.contentRef.current.scrollBy({
            top: 500,
            behavior: "auto",
        });
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

    // Filtered watches - now just returns the watches loaded from search results
    filteredWatches: Atom<CombinedWatchSample[]> = atom((get): CombinedWatchSample[] => {
        const watches = get(this.appRunWatches);
        const pinnedWatchNums = get(this.pinnedWatchNums);

        // Filter out null watches
        const validWatches = watches.filter((watch) => watch != null);

        // Separate pinned and unpinned watches
        const pinnedWatches = validWatches.filter((watch) => pinnedWatchNums.has(watch.watchnum));
        const unpinnedWatches = validWatches.filter((watch) => !pinnedWatchNums.has(watch.watchnum));

        // Sort each group by watch name
        pinnedWatches.sort((a, b) => a.decl.name.localeCompare(b.decl.name));
        unpinnedWatches.sort((a, b) => a.decl.name.localeCompare(b.decl.name));

        // Return pinned watches first, then unpinned
        return [...pinnedWatches, ...unpinnedWatches];
    });

    // Search for watches matching the search term
    async searchWatches(searchTerm: string) {
        const store = getDefaultStore();
        const searchId = crypto.randomUUID();
        this.currentSearchId = searchId;

        try {
            // Call the search RPC to get matching watch IDs
            const searchResult = await RpcApi.WatchSearchRequestCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                searchterm: searchTerm,
            });

            // Check if this search is still the current one
            if (this.currentSearchId !== searchId) {
                return; // Abandon results from stale search
            }

            // Update search result info
            store.set(this.searchResultInfo, {
                searchedCount: searchResult.searchedcount,
                totalCount: searchResult.totalcount,
                errorSpans: searchResult.errorspans || [],
            });

            // Store the matched watch IDs
            const watchIds = searchResult.results;
            store.set(this.matchedWatchIds, watchIds);

            // If we have matching IDs, fetch the watch details
            if (watchIds.length > 0) {
                await this.fetchWatchesByIds(watchIds);
            } else {
                // Clear watches if no matches
                store.set(this.appRunWatches, []);
            }
        } catch (error) {
            console.error(`Failed to search watches for app run ${this.appRunId}:`, error);
            // Reset state on error
            store.set(this.matchedWatchIds, []);
            store.set(this.appRunWatches, []);
            store.set(this.searchResultInfo, { searchedCount: 0, totalCount: 0, errorSpans: [] });
        } finally {
            // No cleanup needed
        }
    }

    // Fetch watch details by IDs
    async fetchWatchesByIds(watchIds: number[]) {
        const searchId = this.currentSearchId;

        try {
            if (watchIds.length === 0) {
                getDefaultStore().set(this.appRunWatches, []);
                return;
            }

            const result = await RpcApi.GetAppRunWatchesByIdsCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                watchids: watchIds,
            });

            // Check if this search is still the current one
            if (this.currentSearchId !== searchId) {
                return; // Abandon results from stale search
            }

            getDefaultStore().set(this.appRunWatches, result.watches);
        } catch (error) {
            console.error(`Failed to fetch watch details for app run ${this.appRunId}:`, error);
            getDefaultStore().set(this.appRunWatches, []);
        }
    }

    // Load watches based on current search term
    async loadAppRunWatches(minTime: number = 0) {
        const startTime = new Date().getTime();
        const store = getDefaultStore();
        const searchTerm = store.get(this.searchTerm);

        try {
            await this.searchWatches(searchTerm);

            // If minTime is specified, ensure we wait at least that long
            if (minTime > 0) {
                const curTime = new Date().getTime();
                if (curTime - startTime < minTime) {
                    await new Promise((r) => setTimeout(r, minTime - (curTime - startTime)));
                }
            }
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
            // Use the current search term to refresh
            const searchTerm = store.get(this.searchTerm);
            await this.searchWatches(searchTerm);
        } catch (error) {
            console.error(`Failed to auto-refresh watches for app run ${this.appRunId}:`, error);
        }
    }

    // Toggle pin status for a watch
    toggleWatchPin(watchNum: number) {
        const store = getDefaultStore();
        const currentPinned = store.get(this.pinnedWatchNums);
        const newPinned = new Set(currentPinned);
        
        if (newPinned.has(watchNum)) {
            newPinned.delete(watchNum);
        } else {
            newPinned.add(watchNum);
        }
        
        store.set(this.pinnedWatchNums, newPinned);
    }

    // Get a derived atom for checking if a specific watch is pinned
    getWatchPinnedAtom(watchNum: number): Atom<boolean> {
        if (!this.pinnedAtomCache.has(watchNum)) {
            const pinnedAtom = atom((get) => {
                const pinnedWatchNums = get(this.pinnedWatchNums);
                return pinnedWatchNums.has(watchNum);
            });
            this.pinnedAtomCache.set(watchNum, pinnedAtom);
        }
        return this.pinnedAtomCache.get(watchNum)!;
    }

    // Update search term and trigger search
    async updateSearchTerm(term: string) {
        const store = getDefaultStore();
        store.set(this.searchTerm, term);
        await this.searchWatches(term);
    }
}

export { WatchesModel };
