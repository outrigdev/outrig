import { DefaultRpcClient } from "@/init";
import { PromiseQueue } from "@/util/promisequeue";
import { Atom, atom, getDefaultStore, Getter, PrimitiveAtom } from "jotai";
import { VirtuosoHandle } from "react-virtuoso";
import { RpcApi } from "../rpc/rpcclientapi";

const PAGESIZE = 100;

type LogCacheEntry = {
    status: "init" | "loading" | "loaded";
    lines: LogLine[];
    version: number;
};

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    searchTerm: PrimitiveAtom<string> = atom("");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isLoading: PrimitiveAtom<boolean> = atom(false);
    followOutput: PrimitiveAtom<boolean> = atom(true);
    virtuosoRef: React.RefObject<VirtuosoHandle> = null;

    totalItemCount: PrimitiveAtom<number> = atom(0);
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

        const cmdPromiseFn = () => {
            return RpcApi.LogSearchRequestCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                apprunid: this.appRunId,
                searchterm: searchTerm,
                requestwindow: { start: 0, size: PAGESIZE },
                stream: false,
            });
        };
        try {
            console.log("searchtermupdate, loading page 0 for search term", searchTerm);
            this.setLogCacheEntry(0, "loading", []);
            const results = await this.requestQueue.enqueue(cmdPromiseFn);
            this.logItemCache = [];
            getDefaultStore().set(this.totalItemCount, results.totalcount);
            getDefaultStore().set(this.filteredItemCount, results.filteredcount);
            getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);
            this.setLogCacheEntry(0, "loaded", results.lines);
        } catch (e) {
            this.setLogCacheEntry(0, "loaded", []);
            console.error("Log search error", e);
        } finally {
            clearTimeout(quickSearchTimeoutId);
            getDefaultStore().set(this.isLoading, false);
            const endTime = performance.now();
            console.log("Log search took", endTime - startTime, "ms", getDefaultStore().get(this.logItemCacheVersion));
        }
    }

    setRenderedRange(start: number, end: number) {
        const startPage = Math.floor(start / PAGESIZE);
        const endPage = Math.floor(end / PAGESIZE);
        for (let i = startPage; i <= endPage; i++) {
            this.fetchLogPage(i);
        }
    }

    getLogItemCacheChunkAtom(page: number, getFn: Getter): PrimitiveAtom<LogCacheEntry> {
        const version = getFn(this.logItemCacheVersion);
        if (!this.logItemCache[page]) {
            const cacheEntry: LogCacheEntry = { status: "init", lines: [], version: version };
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
        const version = getDefaultStore().get(this.logItemCacheVersion);
        const cacheEntry: LogCacheEntry = {
            status: status,
            lines: lines ?? [],
            version: version,
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
                requestwindow: { start, size: PAGESIZE },
                stream: false,
            });
        };
        try {
            console.log("fetchlogpage, loading page " + page + " for search term", searchTerm);
            this.setLogCacheEntry(page, "loading", []);
            const results = await this.requestQueue.enqueue(cmdPromiseFn);
            getDefaultStore().set(this.totalItemCount, results.totalcount);
            getDefaultStore().set(this.filteredItemCount, results.filteredcount);
            this.setLogCacheEntry(page, "loaded", results.lines);
        } catch (e) {
            console.error("Log search error", e);
        }
    }

    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        store.set(this.isRefreshing, true);
        this.logItemCache = [];
        getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);
        try {
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
        if (this.markedLines.has(lineNumber)) {
            this.markedLines.delete(lineNumber);
        } else {
            this.markedLines.add(lineNumber);
        }
        // Increment version to trigger reactivity
        getDefaultStore().set(this.markedLinesVersion, (v) => v + 1);
    }

    isLineMarked(lineNumber: number): boolean {
        return this.markedLines.has(lineNumber);
    }

    clearMarkedLines() {
        this.markedLines.clear();
        // Increment version to trigger reactivity
        getDefaultStore().set(this.markedLinesVersion, (v) => v + 1);
    }

    getMarkedLinesCount(): number {
        return this.markedLines.size;
    }

    // Get all marked lines and extract their messages
    async copyMarkedLinesToClipboard() {
        if (this.markedLines.size === 0) return;

        // Collect all marked line numbers
        const lineNumbers = Array.from(this.markedLines).sort((a, b) => a - b);
        const messages: string[] = [];

        // For each marked line, find the corresponding log line and extract the message
        for (const lineNum of lineNumbers) {
            // Find the page that contains this line
            const page = Math.floor(lineNum / PAGESIZE);

            // Check if we have this page in cache
            if (this.logItemCache[page]) {
                const cacheEntry = getDefaultStore().get(this.logItemCache[page]);
                if (cacheEntry?.lines) {
                    // Find the line in the page
                    const line = cacheEntry.lines.find((l) => l.linenum === lineNum);
                    if (line) {
                        messages.push(line.msg);
                    }
                }
            }
        }

        // Join messages with newlines and copy to clipboard
        if (messages.length > 0) {
            const text = messages.join("");
            await navigator.clipboard.writeText(text);
        }
    }
}

export { LogViewerModel };
