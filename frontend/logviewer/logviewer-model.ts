import { DefaultRpcClient } from "@/init";
import { PromiseQueue } from "@/util/promisequeue";
import { Atom, atom, getDefaultStore, Getter, PrimitiveAtom } from "jotai";
import { unstable_batchedUpdates } from "react-dom";
import { VirtuosoHandle } from "react-virtuoso";
import { RpcApi } from "../rpc/rpcclientapi";

const PAGESIZE = 100;

type LogCacheEntry = {
    status: "init" | "loading" | "loaded";
    lines: LogLine[];
};

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    createTs: number = Date.now();
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isLoading: PrimitiveAtom<boolean> = atom(false);
    followOutput: PrimitiveAtom<boolean> = atom(true);
    virtuosoRef: React.RefObject<VirtuosoHandle> = null;

    // Store the last visible range
    lastVisibleStartIndex: number = 0;
    lastVisibleEndIndex: number = 0;

    totalItemCount: PrimitiveAtom<number> = atom(0);
    searchedItemCount: PrimitiveAtom<number> = atom(0);
    filteredItemCount: PrimitiveAtom<number> = atom(0);

    logItemCacheVersion: PrimitiveAtom<number> = atom(0);
    logItemCache: PrimitiveAtom<LogCacheEntry>[] = [];

    // Store marked lines in a regular Set
    markedLines: Set<number> = new Set<number>();
    // Version atom to trigger reactivity when the set changes
    markedLinesVersion: PrimitiveAtom<number> = atom(0);

    requestQueue: PromiseQueue = new PromiseQueue();
    keepAliveTimeoutId: NodeJS.Timeout = null;

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
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
    }

    async onSearchTermUpdate(searchTerm: string) {
        const startTime = performance.now();
        this.requestQueue.clearQueue();
        const quickSearchTimeoutId = setTimeout(() => {
            getDefaultStore().set(this.isLoading, true);
        }, 200);
        const followOutput = getDefaultStore().get(this.followOutput);

        const visibleItemCount = this.lastVisibleEndIndex - this.lastVisibleStartIndex + 1;
        let pagesNeeded = Math.max(1, Math.ceil(visibleItemCount / PAGESIZE));
        let requestPages: number[];

        if (followOutput) {
            // For follow mode, add 1 to account for potentially partial last page
            pagesNeeded += 1;
            requestPages = Array.from({ length: pagesNeeded }, (_, i) => -pagesNeeded + i);
        } else {
            const startPage = Math.floor(this.lastVisibleStartIndex / PAGESIZE);
            requestPages = Array.from({ length: pagesNeeded }, (_, i) => startPage + i);
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
            this.setLogCacheEntry(0, "loading", []);
            const results = await this.requestQueue.enqueue(cmdPromiseFn);
            console.log("searchresults", results);
            this.logItemCache = [];
            unstable_batchedUpdates(() => {
                getDefaultStore().set(this.totalItemCount, results.totalcount);
                getDefaultStore().set(this.searchedItemCount, results.searchedcount);
                getDefaultStore().set(this.filteredItemCount, results.filteredcount);
                for (let i = 0; i < results.pages.length; i++) {
                    const page = results.pages[i];
                    this.setLogCacheEntry(page.pagenum, "loaded", page.lines ?? []);
                }
                // If following output, scroll to bottom
                if (followOutput) {
                    this.scrollToBottom();
                }
            });
            // Use setTimeout to allow any scrolling to complete
            setTimeout(() => {
                // Fetch the pages for the last visible range
                this.fetchLastVisibleRange();
            }, 10); // Very small delay to allow scrolling to complete
        } catch (e) {
            this.setLogCacheEntry(0, "loaded", []);
            console.error("Log search error", e);
        } finally {
            clearTimeout(quickSearchTimeoutId);
            getDefaultStore().set(this.isLoading, false);
            getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);
            const endTime = performance.now();
            console.log("Log search took", endTime - startTime, "ms", getDefaultStore().get(this.logItemCacheVersion));
        }
    }

    setRenderedRange(start: number, end: number) {
        // Cache the visible range
        this.lastVisibleStartIndex = start;
        this.lastVisibleEndIndex = end;

        if (getDefaultStore().get(this.isLoading)) {
            // If loading, don't fetch pages
            return;
        }

        // virtuoso is slow to update, so we need to do this adjustment
        const followOutput = getDefaultStore().get(this.followOutput);
        if (followOutput) {
            const totalCount = getDefaultStore().get(this.totalItemCount);
            const visibleCount = end - start;
            end = totalCount;
            start = Math.max(0, totalCount - visibleCount);
            this.lastVisibleStartIndex = start;
            this.lastVisibleEndIndex = end;
        }

        const startPage = Math.floor(start / PAGESIZE);
        const endPage = Math.floor(end / PAGESIZE);
        console.log("setRenderedRange", start, end, startPage, endPage);
        for (let i = startPage; i <= endPage; i++) {
            this.fetchLogPage(i);
        }
    }

    // Fetch pages for the last visible range
    fetchLastVisibleRange() {
        if (this.lastVisibleStartIndex >= 0 && this.lastVisibleEndIndex > 0) {
            // Re-use setRenderedRange with the cached values
            this.setRenderedRange(this.lastVisibleStartIndex, this.lastVisibleEndIndex);
        }
    }

    getLogItemCacheChunkAtom(page: number, getFn: Getter): PrimitiveAtom<LogCacheEntry> {
        const version = getFn(this.logItemCacheVersion);
        if (!this.logItemCache[page]) {
            const cacheEntry: LogCacheEntry = { status: "init", lines: [] };
            this.logItemCache[page] = atom(cacheEntry);
        }
        return this.logItemCache[page];
    }

    getLogIndexAtom(index: number): Atom<LogLine> {
        return atom((get) => {
            const page = Math.floor(index / PAGESIZE);
            const pageIndex = index % PAGESIZE;
            const chunkAtom = this.getLogItemCacheChunkAtom(page, get);
            const chunk = get(chunkAtom);
            return chunk?.lines?.[pageIndex];
        });
    }

    setLogCacheEntry(page: number, status: "init" | "loading" | "loaded", lines: LogLine[]) {
        const chunkAtom = this.getLogItemCacheChunkAtom(page, getDefaultStore().get);
        const cacheEntry: LogCacheEntry = {
            status: status,
            lines: lines ?? [],
        };
        getDefaultStore().set(chunkAtom, cacheEntry);
    }

    cacheEntryNeedsLoading(page: number) {
        const chunkAtom = this.getLogItemCacheChunkAtom(page, getDefaultStore().get);
        const cacheEntry = getDefaultStore().get(chunkAtom);
        return cacheEntry?.status === "init";
    }

    async fetchLogPage(page: number) {
        if (!this.cacheEntryNeedsLoading(page)) {
            return;
        }
        const start = page * PAGESIZE;
        const searchTerm = getDefaultStore().get(this.searchTerm);

        const cmdPromiseFn = () => {
            return RpcApi.LogSearchRequestCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                apprunid: this.appRunId,
                searchterm: searchTerm,
                pagesize: PAGESIZE,
                requestpages: [page],
            });
        };
        const startTime = Date.now();
        try {
            console.log("fetchlogpage, loading page " + page + " for search term", searchTerm);
            this.setLogCacheEntry(page, "loading", []);
            const results = await this.requestQueue.enqueue(cmdPromiseFn);
            getDefaultStore().set(this.totalItemCount, results.totalcount);
            getDefaultStore().set(this.searchedItemCount, results.searchedcount);
            getDefaultStore().set(this.filteredItemCount, results.filteredcount);
            // Get lines from the requested page
            const lines = results.pages.find((p) => p.pagenum === page)?.lines || [];
            this.setLogCacheEntry(page, "loaded", lines);
        } catch (e) {
            console.error("Log search error", e);
        } finally {
            console.log("fetchlogpage, loading page " + page + " took", Date.now() - startTime, "ms");
        }
    }

    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        store.set(this.isRefreshing, true);

        try {
            // First, drop the widget to clear the backend cache
            await RpcApi.LogWidgetAdminCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                drop: true,
            });

            // Then clear the frontend cache
            this.logItemCache = [];
            getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);

            // Finally, re-run the search which will create a new SearchManager
            await this.onSearchTermUpdate(store.get(this.searchTerm));
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }

    setVirtuosoRef(ref: React.RefObject<VirtuosoHandle>) {
        this.virtuosoRef = ref;
    }

    pageUp() {
        if (!this.virtuosoRef?.current) return;

        // Virtuoso doesn't have a direct pageUp method, but we can approximate it
        // by scrolling up by a fixed amount
        this.virtuosoRef.current.scrollBy({
            top: -500,
            behavior: "auto",
        });
    }

    pageDown() {
        if (!this.virtuosoRef?.current) return;

        // Virtuoso doesn't have a direct pageDown method, but we can approximate it
        // by scrolling down by a fixed amount
        this.virtuosoRef.current.scrollBy({
            top: 500,
            behavior: "auto",
        });
    }

    scrollToTop() {
        if (!this.virtuosoRef?.current) return;
        this.virtuosoRef.current.scrollToIndex(0);
    }

    scrollToBottom() {
        if (!this.virtuosoRef?.current) return;

        const filteredCount = getDefaultStore().get(this.filteredItemCount);
        if (filteredCount <= 0) return;

        // First scroll to the last item to ensure immediate scroll to bottom
        this.virtuosoRef.current.scrollToIndex({
            index: filteredCount,
            align: "end",
            behavior: "auto",
        });

        // Then set up autoscroll for future updates
        this.virtuosoRef.current.autoscrollToBottom();
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
}

export { LogViewerModel };
