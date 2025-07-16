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
    goroutinestatecounts?: { [key: string]: number };
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

    // Time spans polling state
    timeSpanAtomCache: Map<number, PrimitiveAtom<TimeSpan>> = new Map();
    timeSpansLastTickIdx: number = -1;
    timeSpansPollingInterval: NodeJS.Timeout = null;
    fullTimeSpan: PrimitiveAtom<TimeSpan> = atom<TimeSpan>(null) as PrimitiveAtom<TimeSpan>;
    droppedCount: PrimitiveAtom<number> = atom(0);
    activeCounts: PrimitiveAtom<GoRoutineActiveCount[]> = atom<GoRoutineActiveCount[]>([]);

    // State filters
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Toggle for showing/hiding #outrig goroutines
    showOutrigGoroutines: PrimitiveAtom<boolean> = atom(false);

    // Toggle for showing all goroutines vs only active ones
    showActiveOnly: PrimitiveAtom<boolean> = atom(false);

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

    // Calculate the time offset from the start for display (derived atom)
    timeOffsetSeconds: Atom<number> = atom((get) => {
        const lastSearchTimestamp = get(this.lastSearchTimestamp);
        const fullTimeSpan = get(this.fullTimeSpan);

        if (!fullTimeSpan || lastSearchTimestamp === 0) {
            return 0;
        }

        return Math.floor((lastSearchTimestamp - fullTimeSpan.start) / 1000);
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

        // Sort each group by start time, then by goid if start times are equal
        const sortByStartTime = (a: ParsedGoRoutine, b: ParsedGoRoutine) => {
            const aStart = a.activetimespan?.start || 0;
            const bStart = b.activetimespan?.start || 0;
            if (aStart !== bStart) {
                return aStart - bStart;
            }
            return a.goid - b.goid;
        };

        pinnedGoroutines.sort(sortByStartTime);
        unpinnedGoroutines.sort(sortByStartTime);

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

        this.startTimeSpansPolling();
        this.loadAppRunGoroutines();
    }

    // Clean up resources when component unmounts
    dispose() {
        this.stopTimeSpansPolling();
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
            store.set(this.showActiveOnly, false);
        } else {
            // Replace any existing selection with this state and disable "show all" and "active only"
            store.set(this.selectedStates, new Set([state]));
            store.set(this.showAll, false);
            store.set(this.showActiveOnly, false);
        }

        // Trigger a new search with the current search term to apply the filter
        this.searchGoroutines(store.get(this.searchTerm));
    }

    // Toggle "show all" filter
    toggleShowAll(): void {
        const store = getDefaultStore();
        const showAll = store.get(this.showAll);

        if (!showAll) {
            // If enabling "show all", clear selected states and active only
            store.set(this.selectedStates, new Set<string>());
            store.set(this.showActiveOnly, false);
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

    // Toggle showing all vs active only goroutines
    toggleShowActiveOnly(): void {
        const store = getDefaultStore();
        const showActiveOnly = store.get(this.showActiveOnly);
        
        if (showActiveOnly) {
            // If active is currently on, turn it off and enable show all
            store.set(this.showActiveOnly, false);
            store.set(this.showAll, true);
        } else {
            // If active is off, turn it on and clear other filters
            store.set(this.showActiveOnly, true);
            store.set(this.selectedStates, new Set<string>());
            store.set(this.showAll, false);
        }

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
        const showActiveOnly = store.get(this.showActiveOnly);

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
            // When state filters are active, automatically use activeonly search
            const effectiveActiveOnly = showActiveOnly || selectedStates.size > 0;
            const fullQuery = {
                apprunid: this.appRunId,
                searchterm: searchTerm,
                systemquery: systemQuery,
                timestamp: effectiveTimestamp,
                showoutrig: showOutrig,
                activeonly: effectiveActiveOnly,
            };
            const searchResult = await RpcApi.GoRoutineSearchRequestCommand(DefaultRpcClient, fullQuery);

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
            store.set(this.lastSearchTimestamp, searchResult.effectivesearchtimestamp);

            // Convert int64 IDs to numbers and store them
            const goIds = searchResult.results;
            store.set(this.matchedGoRoutineIds, goIds);

            // If we have matching IDs, fetch the goroutine details
            if (goIds.length > 0) {
                await this.fetchGoRoutinesByIds(goIds, effectiveTimestamp);
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
    async fetchGoRoutinesByIds(goIds: number[], timestamp?: number) {
        const searchId = this.currentSearchId;

        try {
            if (goIds.length === 0) {
                getDefaultStore().set(this.appRunGoRoutines, []);
                return;
            }

            const result = await RpcApi.GetAppRunGoRoutinesByIdsCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                goids: goIds,
                timestamp: timestamp,
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

    // Set the selected timestamp and disable search latest mode
    setSelectedTimestamp(timestamp: number) {
        const store = getDefaultStore();
        // Ensure timestamp is an integer to avoid backend marshalling errors
        const normalizedTimestamp = Math.round(timestamp);
        store.set(this.selectedTimestamp, normalizedTimestamp);
        store.set(this.searchLatestMode, false);
    }

    // Set the selected timestamp and trigger a new search
    setSelectedTimestampAndSearch(timestamp: number) {
        this.setSelectedTimestamp(timestamp);

        // Trigger a new search with the current search term
        const store = getDefaultStore();
        const searchTerm = store.get(this.searchTerm);
        this.searchGoroutines(searchTerm);
    }

    // Enable search latest mode and update to current time
    enableSearchLatest() {
        const store = getDefaultStore();
        const timeSpan = store.get(this.fullTimeSpan);
        const endTime = timeSpan?.end || 0;
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

        return store.get(this.selectedTimestamp) || 0;
    }

    // Get or create a time span atom for a specific goroutine
    getGRTimeSpanAtom(goid: number): PrimitiveAtom<TimeSpan> {
        if (!this.timeSpanAtomCache.has(goid)) {
            const timeSpanAtom = atom<TimeSpan>(null) as PrimitiveAtom<TimeSpan>;
            this.timeSpanAtomCache.set(goid, timeSpanAtom);
        }
        return this.timeSpanAtomCache.get(goid)!;
    }

    // Start polling for goroutine time spans
    startTimeSpansPolling() {
        // Initial call with version 0
        this.pollTimeSpans();

        // Set up interval to poll every second
        this.timeSpansPollingInterval = setInterval(() => {
            this.pollTimeSpans();
        }, 1000);
    }

    // Stop polling for goroutine time spans
    stopTimeSpansPolling() {
        if (this.timeSpansPollingInterval) {
            clearInterval(this.timeSpansPollingInterval);
            this.timeSpansPollingInterval = null;
        }
    }

    // Poll for goroutine time spans using the current version
    async pollTimeSpans() {
        try {
            const response = await RpcApi.GoRoutineTimeSpansCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                sincetickidx: this.timeSpansLastTickIdx,
            });

            // Check if we got a new tick index (new time spans available)
            const hasNewTimeIdx = response.lasttick.idx !== this.timeSpansLastTickIdx;

            // Update version for next call
            this.timeSpansLastTickIdx = response.lasttick.idx;

            // Update individual atoms
            const store = getDefaultStore();

            for (const goTimeSpan of response.data) {
                const timeSpanAtom = this.getGRTimeSpanAtom(goTimeSpan.goid);
                store.set(timeSpanAtom, goTimeSpan.span);
            }

            // Update full time span if it changed
            if (response.fulltimespan) {
                const currentFullTimeSpan = store.get(this.fullTimeSpan);
                if (
                    response.fulltimespan.start !== currentFullTimeSpan?.start ||
                    response.fulltimespan.end !== currentFullTimeSpan?.end
                ) {
                    store.set(this.fullTimeSpan, response.fulltimespan);
                }
            }

            // Update dropped count
            store.set(this.droppedCount, response.droppedcount || 0);

            // Update active counts - append new counts to existing ones
            if (response.activecounts && response.activecounts.length > 0) {
                const currentActiveCounts = store.get(this.activeCounts);
                const updatedActiveCounts = [...currentActiveCounts, ...response.activecounts];
                store.set(this.activeCounts, updatedActiveCounts);
            }

            // If we got new time spans, re-run the search to find new goroutines
            if (hasNewTimeIdx) {
                const searchTerm = store.get(this.searchTerm);
                setTimeout(() => {
                    this.searchGoroutines(searchTerm);
                }, 0);
            }
        } catch (error) {
            console.error(`Failed to poll time spans for app run ${this.appRunId}:`, error);
        }
    }
}

export { GoRoutinesModel };
