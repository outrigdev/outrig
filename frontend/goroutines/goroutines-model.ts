import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

// Type for editor link options
export type CodeLinkType = null | "vscode";

// Type for search result info
export type SearchResultInfo = {
    searchedCount: number;
    totalCount: number;
};

class GoRoutinesModel {
    widgetId: string;
    appRunId: string;
    appRunGoRoutines: PrimitiveAtom<ParsedGoRoutine[]> = atom<ParsedGoRoutine[]>([]);
    matchedGoRoutineIds: PrimitiveAtom<number[]> = atom<number[]>([]);
    searchResultInfo: PrimitiveAtom<SearchResultInfo> = atom<SearchResultInfo>({ searchedCount: 0, totalCount: 0 });
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isSearching: PrimitiveAtom<boolean> = atom(false);
    contentRef: React.RefObject<HTMLDivElement> = null;

    // State filters
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Code link settings
    showCodeLinks: PrimitiveAtom<CodeLinkType> = atom<CodeLinkType>("vscode");

    // Stacktrace display settings - can be "raw", "simplified", or "simplified:files"
    simpleStacktraceMode: PrimitiveAtom<string> = atom("simplified");

    // Total count of goroutines (derived from appRunGoRoutines)
    totalCount: Atom<number> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        return goroutines.length;
    });

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
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

    // Derived atom for primary states
    primaryStates: Atom<string[]> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        const statesSet = new Set<string>();

        goroutines.forEach((goroutine) => {
            if (goroutine.primarystate) {
                statesSet.add(goroutine.primarystate);
            }
        });

        return Array.from(statesSet).sort();
    });

    // Derived atom for extra states
    extraStates: Atom<string[]> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        const statesSet = new Set<string>();

        goroutines.forEach((goroutine) => {
            if (goroutine.extrastates) {
                goroutine.extrastates.forEach((state) => {
                    if (state) {
                        statesSet.add(state);
                    }
                });
            }
        });

        return Array.from(statesSet).sort();
    });

    // Derived atom for duration states, sorted by millisecond value
    durationStates: Atom<string[]> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        // Create a map of duration string to millisecond value
        const durationMap = new Map<string, number>();

        goroutines.forEach((goroutine) => {
            if (goroutine.stateduration && goroutine.statedurationms != null) {
                durationMap.set(goroutine.stateduration, goroutine.statedurationms);
            }
        });

        // Convert to array of [string, number] pairs and sort by millisecond value
        return Array.from(durationMap.entries())
            .sort((a, b) => a[1] - b[1]) // Sort by millisecond value (ascending)
            .map((entry) => entry[0]); // Extract just the duration string
    });

    // Derived atom for all available states (for backward compatibility)
    availableStates: Atom<string[]> = atom((get) => {
        const primaryStates = get(this.primaryStates);
        const extraStates = get(this.extraStates);
        const durationStates = get(this.durationStates);

        return [...primaryStates, ...extraStates, ...durationStates];
    });

    // Derived atom for state counts - returns a map of state name to count
    stateCounts: Atom<Map<string, number>> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        const counts = new Map<string, number>();

        // Initialize counts for all states
        const primaryStates = get(this.primaryStates);
        const extraStates = get(this.extraStates);
        const durationStates = get(this.durationStates);

        [...primaryStates, ...extraStates, ...durationStates].forEach((state) => {
            counts.set(state, 0);
        });

        // Count goroutines for each state
        goroutines.forEach((goroutine) => {
            // Count primary state
            if (goroutine.primarystate) {
                counts.set(goroutine.primarystate, (counts.get(goroutine.primarystate) || 0) + 1);
            }

            // Count extra states
            if (goroutine.extrastates) {
                goroutine.extrastates.forEach((state) => {
                    if (state) {
                        counts.set(state, (counts.get(state) || 0) + 1);
                    }
                });
            }

            // Count duration state
            if (goroutine.stateduration) {
                counts.set(goroutine.stateduration, (counts.get(goroutine.stateduration) || 0) + 1);
            }
        });

        return counts;
    });

    // Toggle a state filter
    toggleStateFilter(state: string): void {
        const store = getDefaultStore();
        const selectedStates = store.get(this.selectedStates);
        const newSelectedStates = new Set(selectedStates);

        if (selectedStates.has(state)) {
            // Remove the state if it's already selected
            newSelectedStates.delete(state);

            // If no states are selected anymore, enable "show all"
            if (newSelectedStates.size === 0) {
                store.set(this.showAll, true);
            }
        } else {
            // Add the state and disable "show all"
            newSelectedStates.add(state);
            store.set(this.showAll, false);
        }

        store.set(this.selectedStates, newSelectedStates);
    }

    // Toggle "show all" filter
    toggleShowAll(): void {
        const store = getDefaultStore();
        const showAll = store.get(this.showAll);

        if (!showAll) {
            // If enabling "show all", clear selected states
            store.set(this.selectedStates, new Set<string>());
        }

        store.set(this.showAll, !showAll);
    }

    // Search for goroutines matching the search term
    async searchGoroutines(searchTerm: string) {
        const store = getDefaultStore();

        try {
            store.set(this.isSearching, true);

            // Call the search RPC to get matching goroutine IDs
            const searchResult = await RpcApi.GoRoutineSearchRequestCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                searchterm: searchTerm,
            });

            // Update search result info
            store.set(this.searchResultInfo, {
                searchedCount: searchResult.searchedcount,
                totalCount: searchResult.totalcount,
            });

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
            store.set(this.searchResultInfo, { searchedCount: 0, totalCount: 0 });
        } finally {
            store.set(this.isSearching, false);
        }
    }

    // Fetch goroutine details by IDs
    async fetchGoRoutinesByIds(goIds: number[]) {
        try {
            if (goIds.length === 0) {
                getDefaultStore().set(this.appRunGoRoutines, []);
                return;
            }

            const result = await RpcApi.GetAppRunGoRoutinesByIdsCommand(DefaultRpcClient, {
                apprunid: this.appRunId,
                goids: goIds,
            });

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

    // Generate a VSCode link for a file path and line number
    generateCodeLink(filePath: string, lineNumber: number, linkType: CodeLinkType): string {
        if (linkType == null) {
            return null;
        }

        if (linkType === "vscode") {
            return `vscode://file${filePath}:${lineNumber}`;
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

    // Update search term and trigger search
    async updateSearchTerm(term: string) {
        const store = getDefaultStore();
        store.set(this.searchTerm, term);
        await this.searchGoroutines(term);
    }
}

export { GoRoutinesModel };
