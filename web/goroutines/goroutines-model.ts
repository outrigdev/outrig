import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

// Type for editor link options
export type CodeLinkType = null | "vscode";

class GoRoutinesModel {
    widgetId: string;
    appRunId: string;
    appRunGoRoutines: PrimitiveAtom<ParsedGoRoutine[]> = atom<ParsedGoRoutine[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);

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

    // Filtered count of goroutines (derived from filteredGoroutines)
    filteredCount: Atom<number> = atom((get) => {
        const filtered = get(this.filteredGoroutines);
        return filtered.length;
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
                goroutine.extrastates.forEach(state => {
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
            .map(entry => entry[0]); // Extract just the duration string
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
        
        [...primaryStates, ...extraStates, ...durationStates].forEach(state => {
            counts.set(state, 0);
        });
        
        // Count goroutines for each state
        goroutines.forEach(goroutine => {
            // Count primary state
            if (goroutine.primarystate) {
                counts.set(goroutine.primarystate, (counts.get(goroutine.primarystate) || 0) + 1);
            }
            
            // Count extra states
            if (goroutine.extrastates) {
                goroutine.extrastates.forEach(state => {
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

    // Filtered goroutines based on search term and state filters
    filteredGoroutines: Atom<ParsedGoRoutine[]> = atom((get) => {
        const search = get(this.searchTerm);
        const showAll = get(this.showAll);
        const selectedStates = get(this.selectedStates);
        const goroutines = get(this.appRunGoRoutines);
        const durationStates = get(this.durationStates);

        // First sort by goroutine ID
        const sortedGoroutines = [...goroutines].sort((a, b) => a.goid - b.goid);

        // Apply state filters if not showing all
        let stateFiltered = sortedGoroutines;
        if (!showAll && selectedStates.size > 0) {
            // Get the selected duration states and regular states separately
            const selectedDurationStates = new Set<string>();
            const selectedRegularStates = new Set<string>();
            
            selectedStates.forEach(state => {
                if (durationStates.includes(state)) {
                    selectedDurationStates.add(state);
                } else {
                    selectedRegularStates.add(state);
                }
            });

            stateFiltered = sortedGoroutines.filter((goroutine) => {
                // Split the rawstate by commas and get all states for this goroutine
                const states = goroutine.rawstate.split(",").map((s) => s.trim());
                
                // If no regular states are selected, consider it a match for regular states
                // If regular states are selected, at least one must match (OR)
                const matchesRegularStates = selectedRegularStates.size === 0 || 
                    states.some((state) => selectedRegularStates.has(state));
                
                // If no duration states are selected, consider it a match for duration states
                // If duration states are selected, at least one must match (OR)
                const matchesDurationStates = selectedDurationStates.size === 0 || 
                    (goroutine.stateduration && selectedDurationStates.has(goroutine.stateduration));
                
                // Both conditions must be true (AND)
                return matchesRegularStates && matchesDurationStates;
            });
        }

        // Apply search filter if there's a search term
        if (!search) {
            return stateFiltered;
        }

        return stateFiltered.filter(
            (goroutine) =>
                goroutine.rawstacktrace.toLowerCase().includes(search.toLowerCase()) ||
                goroutine.rawstate.toLowerCase().includes(search.toLowerCase()) ||
                goroutine.goid.toString().includes(search)
        );
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

    async fetchAppRunGoroutines() {
        try {
            const result = await RpcApi.GetAppRunGoRoutinesCommand(DefaultRpcClient, { apprunid: this.appRunId });
            return result.goroutines;
        } catch (error) {
            console.error(`Failed to load goroutines for app run ${this.appRunId}:`, error);
            return [];
        }
    }

    // Load goroutines with a minimum time to show the refreshing state
    async loadAppRunGoroutines(minTime: number = 0) {
        const startTime = new Date().getTime();

        try {
            const goroutines = await this.fetchAppRunGoroutines();

            // If minTime is specified, ensure we wait at least that long
            if (minTime > 0) {
                const curTime = new Date().getTime();
                if (curTime - startTime < minTime) {
                    await new Promise((r) => setTimeout(r, minTime - (curTime - startTime)));
                }
            }

            getDefaultStore().set(this.appRunGoRoutines, goroutines);
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
}

export { GoRoutinesModel };
