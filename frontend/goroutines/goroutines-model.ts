// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { SearchStore } from "@/store/searchstore";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

// Maximum time range for the slider (10 minutes minus small buffer for valid searches)
const MAX_TIME_RANGE_SECONDS = 600 - 5;

// Type for search result info
export type SearchResultInfo = {
    searchedCount: number;
    totalCount: number;
    totalnonoutrig?: number;
    goroutinestatecounts?: {[key: string]: number};
    errorSpans?: SearchErrorSpan[];
};

class GoRoutinesModel {
    widgetId: string;
    appRunId: string;
    appRunGoRoutines: PrimitiveAtom<ParsedGoRoutine[]> = atom<ParsedGoRoutine[]>([]);
    matchedGoRoutineIds: PrimitiveAtom<number[]> = atom<number[]>([]);
    pinnedGoRoutineIds: PrimitiveAtom<Set<number>> = atom<Set<number>>(new Set<number>());
    searchResultInfo: PrimitiveAtom<SearchResultInfo> = atom<SearchResultInfo>({
        searchedCount: 0,
        totalCount: 0,
        errorSpans: [],
    });
    searchTerm: PrimitiveAtom<string>;
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    lastSearchTimestamp: PrimitiveAtom<number> = atom(0);
    selectedTimestamp: PrimitiveAtom<number> = atom(0);
    searchLatestMode: PrimitiveAtom<boolean> = atom(true);
    contentRef: React.RefObject<HTMLDivElement> = null;
    currentSearchId: string = "";
    pinnedAtomCache: Map<number, Atom<boolean>> = new Map();

    // State filters
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Toggle for showing/hiding #outrig goroutines
    showOutrigGoroutines: PrimitiveAtom<boolean> = atom(false);

    // Stacktrace display settings - can be "raw", "simplified", or "simplified:files"
    simpleStacktraceMode: PrimitiveAtom<string> = atom("simplified");

    // Effective stacktrace mode that considers search term
    // Returns "raw" when search is active, otherwise returns the user-selected mode
    effectiveSimpleStacktraceMode: Atom<string> = atom((get) => {
        const searchTerm = get(this.searchTerm);
        const userSelectedMode = get(this.simpleStacktraceMode);

        // If there's a search term, use raw mode to make matches visible
        if (searchTerm && searchTerm.trim() !== "") {
            return "raw";
        }

        // Otherwise use the user-selected mode
        return userSelectedMode;
    });

    // Total count of goroutines (derived from appRunGoRoutines)
    totalCount: Atom<number> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        return goroutines.length;
    });

    // Actual result count (derived from matchedGoRoutineIds)
    resultCount: Atom<number> = atom((get) => {
        const matchedIds = get(this.matchedGoRoutineIds);
        return matchedIds.length;
    });

    // Sorted goroutines with pinned ones first
    sortedGoRoutines: Atom<ParsedGoRoutine[]> = atom((get): ParsedGoRoutine[] => {
        const goroutines = get(this.appRunGoRoutines);
        const pinnedGoRoutineIds = get(this.pinnedGoRoutineIds);

        // Separate pinned and unpinned goroutines
        const pinnedGoroutines = goroutines.filter((gr) => pinnedGoRoutineIds.has(gr.goid));
        const unpinnedGoroutines = goroutines.filter((gr) => !pinnedGoRoutineIds.has(gr.goid));

        // Sort each group by goid
        pinnedGoroutines.sort((a, b) => a.goid - b.goid);
        unpinnedGoroutines.sort((a, b) => a.goid - b.goid);

        // Combine with pinned first
        return [...pinnedGoroutines, ...unpinnedGoroutines];
    });

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;

        // Get app name from AppModel using the appRunId
        const appRunInfoAtom = AppModel.getAppRunInfoAtom(appRunId);
        const appRunInfo = getDefaultStore().get(appRunInfoAtom);
        const appName = appRunInfo?.appname || "unknown";

        // Get search term atom from SearchStore
        this.searchTerm = SearchStore.getSearchTermAtom(appName, appRunId, "goroutines");

        this.loadAppRunGoroutines();
    }

    // Clean up resources when component unmounts
    dispose() {
        // Currently no resources to clean up, but this method is added
        // for consistency with other models and future-proofing
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


    // Toggle a state filter (single selection mode)
    toggleStateFilter(state: string): void {
        const store = getDefaultStore();
        const selectedStates = store.get(this.selectedStates);

        if (selectedStates.has(state)) {
            // If clicking the already selected state, deselect it and enable "show all"
            store.set(this.selectedStates, new Set<string>());
            store.set(this.showAll, true);
        } else {
            // Replace any existing selection with this state and disable "show all"
            store.set(this.selectedStates, new Set([state]));
            store.set(this.showAll, false);
        }

        // Trigger a new search with the current search term to apply the filter
        this.searchGoroutines(store.get(this.searchTerm));
    }

    // Toggle "show all" filter
    toggleShowAll(): void {
        const store = getDefaultStore();
        const showAll = store.get(this.showAll);

        if (!showAll) {
            // If enabling "show all", clear selected states
            store.set(this.selectedStates, new Set<string>());
            // Note: We do NOT reset showOutrigGoroutines here
        }

        store.set(this.showAll, !showAll);

        // Trigger a new search with the current search term to apply the filter
        this.searchGoroutines(store.get(this.searchTerm));
    }

    // Toggle showing/hiding #outrig goroutines
    toggleShowOutrigGoroutines(): void {
        const store = getDefaultStore();
        const showOutrig = store.get(this.showOutrigGoroutines);
        store.set(this.showOutrigGoroutines, !showOutrig);

        // Trigger a new search with the current search term
        this.searchGoroutines(store.get(this.searchTerm));
    }

    // Search for goroutines matching the search term
    async searchGoroutines(searchTerm: string) {
        const store = getDefaultStore();
        const searchId = crypto.randomUUID();
        this.currentSearchId = searchId;
        const showOutrig = store.get(this.showOutrigGoroutines);
        const selectedStates = store.get(this.selectedStates);

        try {
            // Build the systemQuery based on selected states and showOutrig setting
            let systemQuery: string | undefined;

            // Start with the base query parts
            const outrigPart = !showOutrig ? "-#outrig" : "";
            const userQueryPart = "#userquery";

            // Handle state filters
            let statesPart = "";
            if (selectedStates.size > 0) {
                const statesArray = Array.from(selectedStates);

                if (statesArray.length === 1) {
                    // Single state filter
                    statesPart = `"${statesArray[0]}"`;
                } else if (statesArray.length > 1) {
                    // Multiple state filters with OR logic
                    const statesString = statesArray.map((state) => `"${state}"`).join(" ");
                    statesPart = `(${statesString})`;
                }
            }

            // Combine the parts to form the final query
            if (outrigPart || statesPart || userQueryPart) {
                const parts = [outrigPart, statesPart, userQueryPart].filter(Boolean);
                systemQuery = parts.join(" ");
            }

            // Get the effective timestamp for the search
            const effectiveTimestamp = this.getEffectiveTimestamp();

            // Call the search RPC to get matching goroutine IDs
            const searchResult = await RpcApi.GoRoutineSearchRequestCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                searchterm: searchTerm,
                systemquery: systemQuery,
                timestamp: effectiveTimestamp > 0 ? effectiveTimestamp : undefined,
                showoutrig: showOutrig,
            });

            // Check if this search is still the current one
            if (this.currentSearchId !== searchId) {
                return; // Abandon results from stale search
            }

            // Update search result info and timestamp
            store.set(this.searchResultInfo, {
                searchedCount: searchResult.searchedcount,
                totalCount: searchResult.totalcount,
                totalnonoutrig: searchResult.totalnonoutrig,
                goroutinestatecounts: searchResult.goroutinestatecounts,
                errorSpans: searchResult.errorspans || [],
            });
            // Set the timestamp to the actual timestamp that was searched for
            let searchedTimestamp: number;
            if (effectiveTimestamp > 0) {
                searchedTimestamp = effectiveTimestamp;
            } else {
                // When effectiveTimestamp is 0 (search latest), use the app run's lastmodtime
                const appRunInfoAtom = AppModel.getAppRunInfoAtom(this.appRunId);
                const appRunInfo = store.get(appRunInfoAtom);
                searchedTimestamp = appRunInfo?.lastmodtime || 0;
            }
            store.set(this.lastSearchTimestamp, searchedTimestamp);

            // Convert int64 IDs to numbers and store them
            const goIds = searchResult.results;
            store.set(this.matchedGoRoutineIds, goIds);

            // If we have matching IDs, fetch the goroutine details
            if (goIds.length > 0) {
                await this.fetchGoRoutinesByIds(goIds);
            } else {
                // Clear goroutines if no matches
                store.set(this.appRunGoRoutines, []);
            }
        } catch (error) {
            console.error(`Failed to search goroutines for app run ${this.appRunId}:`, error);
            // Reset state on error
            store.set(this.matchedGoRoutineIds, []);
            store.set(this.appRunGoRoutines, []);
            store.set(this.searchResultInfo, { searchedCount: 0, totalCount: 0, errorSpans: [] });
        } finally {
            // No cleanup needed
        }
    }

    // Fetch goroutine details by IDs
    async fetchGoRoutinesByIds(goIds: number[]) {
        const searchId = this.currentSearchId;

        try {
            if (goIds.length === 0) {
                getDefaultStore().set(this.appRunGoRoutines, []);
                return;
            }

            const result = await RpcApi.GetAppRunGoRoutinesByIdsCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                goids: goIds,
            });

            // Check if this search is still the current one
            if (this.currentSearchId !== searchId) {
                return; // Abandon results from stale search
            }

            getDefaultStore().set(this.appRunGoRoutines, result.goroutines);
        } catch (error) {
            console.error(`Failed to fetch goroutine details for app run ${this.appRunId}:`, error);
            getDefaultStore().set(this.appRunGoRoutines, []);
        }
    }

    // Load goroutines based on current search term
    async loadAppRunGoroutines(minTime: number = 0) {
        const startTime = new Date().getTime();
        const store = getDefaultStore();
        const searchTerm = store.get(this.searchTerm);

        try {
            await this.searchGoroutines(searchTerm);

            // If minTime is specified, ensure we wait at least that long
            if (minTime > 0) {
                const curTime = new Date().getTime();
                if (curTime - startTime < minTime) {
                    await new Promise((r) => setTimeout(r, minTime - (curTime - startTime)));
                }
            }
        } catch (error) {
            console.error(`Failed to load goroutines for app run ${this.appRunId}:`, error);
        }
    }

    // Parse a stacktrace line to extract file path and line number
    // Example line: "  /Users/mike/work/outrig/server/main-server.go:291 +0x1a5"
    parseStacktraceLine(line: string): { filePath: string; lineNumber: number } {
        // Match a pattern like "/path/to/file.go:123"
        const match = line.match(/(\S+\.go):(\d+)/);
        if (match) {
            return {
                filePath: match[1],
                lineNumber: parseInt(match[2], 10),
            };
        }
        return null;
    }

    // Refresh goroutines with a minimum time to show the refreshing state
    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Set refreshing state to true
        store.set(this.isRefreshing, true);

        // Clear goroutines immediately
        store.set(this.appRunGoRoutines, []);

        try {
            // Load new goroutines with a minimum time of 500ms to show the refreshing state
            await this.loadAppRunGoroutines(500);
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }

    // Toggle pin status for a goroutine
    toggleGoRoutinePin(goid: number) {
        const store = getDefaultStore();
        const currentPinned = store.get(this.pinnedGoRoutineIds);
        const newPinned = new Set(currentPinned);

        if (newPinned.has(goid)) {
            newPinned.delete(goid);
        } else {
            newPinned.add(goid);
        }

        store.set(this.pinnedGoRoutineIds, newPinned);
    }

    // Get a derived atom for checking if a specific goroutine is pinned
    getGoRoutinePinnedAtom(goid: number): Atom<boolean> {
        if (!this.pinnedAtomCache.has(goid)) {
            const pinnedAtom = atom((get) => {
                const pinnedGoRoutineIds = get(this.pinnedGoRoutineIds);
                return pinnedGoRoutineIds.has(goid);
            });
            this.pinnedAtomCache.set(goid, pinnedAtom);
        }
        return this.pinnedAtomCache.get(goid)!;
    }

    // Update search term and trigger search
    async updateSearchTerm(term: string) {
        const store = getDefaultStore();
        store.set(this.searchTerm, term);
        await this.searchGoroutines(term);
    }

    // Get time range for the slider based on app run info
    getTimeRange(): { startTime: number; endTime: number; maxRange: number } {
        const store = getDefaultStore();
        const appRunInfoAtom = AppModel.getAppRunInfoAtom(this.appRunId);
        const appRunInfo = store.get(appRunInfoAtom);

        if (!appRunInfo || !appRunInfo.firstgoroutinecollectionts) {
            return { startTime: 0, endTime: 0, maxRange: MAX_TIME_RANGE_SECONDS };
        }

        const startTime = appRunInfo.firstgoroutinecollectionts;
        const endTime = appRunInfo.lastmodtime;

        // Calculate actual duration in seconds
        const actualDurationSeconds = Math.floor((endTime - startTime) / 1000);

        // If the actual duration is less than our max range, use the actual duration
        if (actualDurationSeconds <= MAX_TIME_RANGE_SECONDS) {
            return {
                startTime,
                endTime,
                maxRange: actualDurationSeconds,
            };
        }

        // If duration exceeds max range, adjust start time and use max range
        const adjustedStartTime = endTime - MAX_TIME_RANGE_SECONDS * 1000;
        return {
            startTime: adjustedStartTime,
            endTime,
            maxRange: MAX_TIME_RANGE_SECONDS,
        };
    }

    // Set the selected timestamp and disable search latest mode
    setSelectedTimestamp(timestamp: number) {
        const store = getDefaultStore();
        store.set(this.selectedTimestamp, timestamp);
        store.set(this.searchLatestMode, false);
    }

    // Enable search latest mode and update to current time
    enableSearchLatest() {
        const store = getDefaultStore();
        const { endTime } = this.getTimeRange();
        store.set(this.selectedTimestamp, endTime);
        store.set(this.searchLatestMode, true);
    }

    // Get the effective timestamp for searches (0 means latest)
    getEffectiveTimestamp(): number {
        const store = getDefaultStore();
        const searchLatest = store.get(this.searchLatestMode);

        if (searchLatest) {
            return 0; // 0 means use latest timestamp
        }

        return store.get(this.selectedTimestamp);
    }
}

export { GoRoutinesModel };
