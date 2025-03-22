import { DefaultRpcClient } from "@/init";
import { LOG_STREAM_UPDATE_EVENT } from "@/rpcclientimpl";
import { PromiseQueue } from "@/util/promisequeue";
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { selectAtom } from "jotai/utils";
import { unstable_batchedUpdates } from "react-dom";
import { RpcApi } from "../rpc/rpcclientapi";

const PAGESIZE = 100;

// Interfaces moved from logvlist
export interface LogPageInterface {
    lines: LogLine[];
    totalCount: number;
    loaded: boolean;
}

export interface LogListInterface {
    totalCount: number;
    pageSize: number;
    pages: PrimitiveAtom<LogPageInterface>[];
    version: number;
}

// Interface for log counts
interface LogCounts {
    total: number;
    searched: number;
    filtered: number;
}

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    createTs: number = Date.now();
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isLoading: PrimitiveAtom<boolean> = atom(false);
    followOutput: PrimitiveAtom<boolean> = atom(true);
    vlistRef: React.RefObject<HTMLDivElement> = { current: null };

    // LogVList state
    listAtom: PrimitiveAtom<LogListInterface>;
    listVersion: number = 0;

    // Single atom to hold all count values
    logCountsAtom: PrimitiveAtom<LogCounts> = atom<LogCounts>({
        total: 0,
        searched: 0,
        filtered: 0,
    });

    // Derived atoms for individual counts (read-only)
    totalItemCount = selectAtom(this.logCountsAtom, (state) => state.total);
    searchedItemCount = selectAtom(this.logCountsAtom, (state) => state.searched);
    filteredItemCount = selectAtom(this.logCountsAtom, (state) => state.filtered);

    // Store marked lines in a regular Set
    markedLines: Set<number> = new Set<number>();
    // Version atom to trigger reactivity when the set changes
    markedLinesVersion: PrimitiveAtom<number> = atom(0);

    requestQueue: PromiseQueue = new PromiseQueue();
    keepAliveTimeoutId: NodeJS.Timeout = null;

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;

        // Initialize the list atom with empty state
        this.listAtom = atom<LogListInterface>({
            totalCount: 0,
            pageSize: PAGESIZE,
            pages: [],
            version: 0,
        });

        this.keepAliveTimeoutId = setInterval(() => {
            RpcApi.LogWidgetAdminCommand(
                DefaultRpcClient,
                {
                    widgetid: this.widgetId,
                    keepalive: true,
                },
                { noresponse: true }
            );
        }, 5000);
        
        // Register for log stream update events
        document.addEventListener(LOG_STREAM_UPDATE_EVENT, this.handleLogStreamUpdate as EventListener);
    }

    dispose() {
        clearInterval(this.keepAliveTimeoutId);
        RpcApi.LogWidgetAdminCommand(
            DefaultRpcClient,
            {
                widgetid: this.widgetId,
                drop: true,
            },
            { noresponse: true }
        );

        // Clean up event listener
        document.removeEventListener(LOG_STREAM_UPDATE_EVENT, this.handleLogStreamUpdate as EventListener);
    }

    async onSearchTermUpdate(searchTerm: string) {
        const startTime = performance.now();
        this.requestQueue.clearQueue();
        const quickSearchTimeoutId = setTimeout(() => {
            getDefaultStore().set(this.isLoading, true);
        }, 200);
        const followOutput = getDefaultStore().get(this.followOutput);

        // Request initial pages
        let requestPages: number[];

        if (followOutput) {
            // In follow mode, request 3 pages (last page and 2 preceding pages)
            requestPages = [-3, -2, -1];
        } else {
            // In non-follow mode, request first 2 pages
            requestPages = [0, 1];
        }

        const cmdPromiseFn = () => {
            return RpcApi.LogSearchRequestCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                apprunid: this.appRunId,
                searchterm: searchTerm,
                pagesize: PAGESIZE,
                requestpages: requestPages,
            });
        };

        try {
            console.log(
                "searchtermupdate, loading results for search term",
                searchTerm,
                "@" + (Date.now() - this.createTs) + "ms"
            );

            const results = await this.requestQueue.enqueue(cmdPromiseFn);
            console.log("searchresults", results);

            // Increment version to trigger a full reset
            this.listVersion++;

            // Calculate total number of pages needed
            const totalPages = Math.ceil(results.filteredcount / PAGESIZE);

            // Create page atoms for all pages (most will be unloaded)
            const pageAtoms: PrimitiveAtom<LogPageInterface>[] = [];

            for (let i = 0; i < totalPages; i++) {
                // Find if this page was loaded in the results
                const loadedPage = results.pages.find((p) => p.pagenum === i);

                if (loadedPage) {
                    // This page was loaded in the results
                    pageAtoms[i] = atom<LogPageInterface>({
                        lines: loadedPage.lines || [],
                        totalCount:
                            i === totalPages - 1 ? results.filteredcount - (totalPages - 1) * PAGESIZE : PAGESIZE,
                        loaded: true,
                    });
                } else {
                    // This page needs to be loaded on demand
                    const itemsInPage =
                        i === totalPages - 1 ? results.filteredcount - (totalPages - 1) * PAGESIZE : PAGESIZE;

                    pageAtoms[i] = atom<LogPageInterface>({
                        lines: [],
                        totalCount: itemsInPage,
                        loaded: false,
                    });
                }
            }

            // Update the list atom with the new state
            unstable_batchedUpdates(() => {
                getDefaultStore().set(this.logCountsAtom, {
                    total: results.totalcount,
                    searched: results.searchedcount,
                    filtered: results.filteredcount,
                });

                getDefaultStore().set(this.listAtom, {
                    totalCount: results.filteredcount,
                    pageSize: PAGESIZE,
                    pages: pageAtoms,
                    version: this.listVersion,
                });
            });
        } catch (e) {
            console.error("Log search error", e);

            // Reset to empty state on error
            unstable_batchedUpdates(() => {
                getDefaultStore().set(this.logCountsAtom, {
                    total: 0,
                    searched: 0,
                    filtered: 0,
                });

                getDefaultStore().set(this.listAtom, {
                    totalCount: 0,
                    pageSize: PAGESIZE,
                    pages: [],
                    version: this.listVersion,
                });
            });
        } finally {
            clearTimeout(quickSearchTimeoutId);
            getDefaultStore().set(this.isLoading, false);
            const endTime = performance.now();
            console.log("Log search took", endTime - startTime, "ms");
        }
    }

    // Handle page required callback from LogVList
    async onPageRequired(pageNum: number, load: boolean = true) {
        console.log("Page required:", pageNum, "load:", load);

        // Get the current list state
        const listState = getDefaultStore().get(this.listAtom);

        // Check if this page exists
        if (pageNum >= listState.pages.length) {
            console.error("Page number out of bounds:", pageNum, listState.pages.length);
            return;
        }

        // Get the page atom
        const pageAtom = listState.pages[pageNum];
        const pageState = getDefaultStore().get(pageAtom);

        // If load is false, we should drop the page if it's loaded
        if (!load) {
            if (pageState.loaded) {
                console.log("Dropping page", pageNum);
                // Update the page atom to mark it as unloaded, but keep the totalCount
                getDefaultStore().set(pageAtom, {
                    lines: [],
                    totalCount: pageState.totalCount,
                    loaded: false,
                });
            }
            return;
        }

        // If already loaded or loading, do nothing
        if (pageState.loaded) {
            return;
        }

        // Get the search term
        const searchTerm = getDefaultStore().get(this.searchTerm);

        const cmdPromiseFn = () => {
            return RpcApi.LogSearchRequestCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                apprunid: this.appRunId,
                searchterm: searchTerm,
                pagesize: PAGESIZE,
                requestpages: [pageNum],
            });
        };

        const startTime = Date.now();
        try {
            console.log("Loading page", pageNum, "for search term", searchTerm);

            const results = await this.requestQueue.enqueue(cmdPromiseFn);

            // Get lines from the requested page
            const loadedPage = results.pages.find((p) => p.pagenum === pageNum);

            if (loadedPage) {
                // Update just this page atom
                getDefaultStore().set(pageAtom, {
                    lines: loadedPage.lines || [],
                    totalCount: pageState.totalCount,
                    loaded: true,
                });

                // Also update the counts in case they changed
                getDefaultStore().set(this.logCountsAtom, {
                    total: results.totalcount,
                    searched: results.searchedcount,
                    filtered: results.filteredcount,
                });
            }
        } catch (e) {
            console.error("Log page load error", e);
        } finally {
            console.log("Loading page", pageNum, "took", Date.now() - startTime, "ms");
        }
    }

    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Record start time to ensure minimum display duration
        const startTime = Date.now();
        store.set(this.isRefreshing, true);

        try {
            // First, drop the widget to clear the backend cache
            await RpcApi.LogWidgetAdminCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                drop: true,
            });

            // Then re-run the search which will create a new SearchManager and list atom
            await this.onSearchTermUpdate(store.get(this.searchTerm));
        } finally {
            // Calculate elapsed time
            const elapsedTime = Date.now() - startTime;
            const remainingTime = Math.max(0, 500 - elapsedTime);

            // If less than 1000ms has passed, wait for the remainder
            if (remainingTime > 0) {
                await new Promise((resolve) => setTimeout(resolve, remainingTime));
            }

            // Set refreshing state to false after ensuring minimum display time
            store.set(this.isRefreshing, false);
        }
    }

    // Scroll control methods for LogVList
    scrollToTop() {
        if (!this.vlistRef?.current) return;
        this.vlistRef.current.scrollTop = 0;
    }

    scrollToBottom() {
        if (!this.vlistRef?.current) return;
        this.vlistRef.current.scrollTop = this.vlistRef.current.scrollHeight;
    }

    pageUp() {
        if (!this.vlistRef?.current) return;
        this.vlistRef.current.scrollBy({
            top: -this.vlistRef.current.clientHeight,
            behavior: "auto",
        });
    }

    pageDown() {
        if (!this.vlistRef?.current) return;
        this.vlistRef.current.scrollBy({
            top: this.vlistRef.current.clientHeight,
            behavior: "auto",
        });
    }

    setVListRef(ref: React.RefObject<HTMLDivElement>) {
        this.vlistRef = ref;
    }

    // Methods for managing marked lines
    toggleLineMarked(lineNumber: number) {
        const isMarked = this.markedLines.has(lineNumber);

        if (isMarked) {
            this.markedLines.delete(lineNumber);
        } else {
            this.markedLines.add(lineNumber);
        }

        // Increment version to trigger reactivity
        getDefaultStore().set(this.markedLinesVersion, (v) => v + 1);

        // Send just the delta to the backend
        const markedLinesMap: Record<string, boolean> = {};
        markedLinesMap[lineNumber.toString()] = !isMarked;

        RpcApi.LogUpdateMarkedLinesCommand(
            DefaultRpcClient,
            {
                widgetid: this.widgetId,
                markedlines: markedLinesMap,
                clear: false,
            },
            { noresponse: true }
        );
    }

    isLineMarked(lineNumber: number): boolean {
        return this.markedLines.has(lineNumber);
    }

    clearMarkedLines() {
        this.markedLines.clear();
        // Increment version to trigger reactivity
        getDefaultStore().set(this.markedLinesVersion, (v) => v + 1);

        // Send clear command to the backend
        RpcApi.LogUpdateMarkedLinesCommand(
            DefaultRpcClient,
            {
                widgetid: this.widgetId,
                markedlines: {},
                clear: true,
            },
            { noresponse: true }
        );

        // Refresh search results after clearing all marked lines
        this.refresh();
    }

    getMarkedLinesCount(): number {
        return this.markedLines.size;
    }

    // Get all marked lines from the backend and copy their messages to clipboard
    async copyMarkedLinesToClipboard() {
        if (this.markedLines.size === 0) return;

        try {
            // Request marked lines from the backend
            const result = await RpcApi.LogGetMarkedLinesCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
            });

            if (!result.lines || result.lines.length === 0) {
                console.log("No marked lines returned from backend");
                return;
            }

            // Extract messages
            const messages = result.lines.map((line: LogLine) => line.msg);

            // Join messages and copy to clipboard
            const text = messages.join("");
            await navigator.clipboard.writeText(text);
        } catch (error) {
            console.error("Failed to get marked lines from backend:", error);
        }
    }

    // Handler for log stream updates
    handleLogStreamUpdate = (event: CustomEvent<StreamUpdateData>) => {
        // Stub implementation - will be filled in later
        console.log("Log stream update received:", event.detail);
    };
}

export { LogViewerModel };
