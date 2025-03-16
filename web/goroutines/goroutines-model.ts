import { DefaultRpcClient } from "@/init";
import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcApi } from "../rpc/rpcclientapi";

// Type for editor link options
export type CodeLinkType = null | "vscode";

class GoRoutinesModel {
    widgetId: string;
    appRunId: string;
    appRunGoRoutines: PrimitiveAtom<GoroutineData[]> = atom<GoroutineData[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);

    // State filters
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Code link settings
    showCodeLinks: PrimitiveAtom<CodeLinkType> = atom<CodeLinkType>("vscode");

    // Stacktrace display settings
    simpleStacktraceMode: PrimitiveAtom<boolean> = atom(true);

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

    // Derived atom for all available states
    availableStates: Atom<string[]> = atom((get) => {
        const goroutines = get(this.appRunGoRoutines);
        const statesSet = new Set<string>();

        goroutines.forEach((goroutine) => {
            statesSet.add(goroutine.state);
        });

        return Array.from(statesSet).sort();
    });

    // Filtered goroutines based on search term and state filters
    filteredGoroutines: Atom<GoroutineData[]> = atom((get) => {
        const search = get(this.searchTerm);
        const showAll = get(this.showAll);
        const selectedStates = get(this.selectedStates);
        const goroutines = get(this.appRunGoRoutines);

        // First sort by goroutine ID
        const sortedGoroutines = [...goroutines].sort((a, b) => a.goid - b.goid);

        // Apply state filters if not showing all
        let stateFiltered = sortedGoroutines;
        if (!showAll && selectedStates.size > 0) {
            stateFiltered = sortedGoroutines.filter((goroutine) => selectedStates.has(goroutine.state));
        }

        // Apply search filter if there's a search term
        if (!search) {
            return stateFiltered;
        }

        return stateFiltered.filter(
            (goroutine) =>
                goroutine.stacktrace.toLowerCase().includes(search.toLowerCase()) ||
                goroutine.state.toLowerCase().includes(search.toLowerCase()) ||
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
            const result = await RpcApi.GetAppRunGoroutinesCommand(DefaultRpcClient, { apprunid: this.appRunId });
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
