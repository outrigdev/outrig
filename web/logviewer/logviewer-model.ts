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

export type SearchType = "exact" | "exactcase" | "regexp" | "fzf";

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    searchTerm: PrimitiveAtom<string> = atom("");
    searchType: PrimitiveAtom<SearchType> = atom<SearchType>("exact");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isLoading: PrimitiveAtom<boolean> = atom(false);
    followOutput: PrimitiveAtom<boolean> = atom(true);
    virtuosoRef: React.RefObject<VirtuosoHandle> = null;

    totalItemCount: PrimitiveAtom<number> = atom(0);
    filteredItemCount: PrimitiveAtom<number> = atom(0);

    logItemCacheVersion: PrimitiveAtom<number> = atom(0);
    logItemCache: PrimitiveAtom<LogCacheEntry>[] = [];

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

    async onSearchTermUpdate(searchTerm: string, searchType: SearchType) {
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
                searchtype: searchType,
                requestwindow: { start: 0, size: PAGESIZE },
                stream: false,
            });
        };
        try {
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
        const searchType = getDefaultStore().get(this.searchType);

        const cmdPromiseFn = () => {
            return RpcApi.LogSearchRequestCommand(DefaultRpcClient, {
                widgetid: this.widgetId,
                apprunid: this.appRunId,
                searchterm: searchTerm,
                searchtype: searchType,
                requestwindow: { start, size: PAGESIZE },
                stream: false,
            });
        };
        try {
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
            await this.onSearchTermUpdate(store.get(this.searchTerm), store.get(this.searchType));
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
            behavior: 'auto'
        });
    }

    pageDown() {
        if (!this.virtuosoRef?.current) return;
        
        // Virtuoso doesn't have a direct pageDown method, but we can approximate it
        // by scrolling down by a fixed amount
        this.virtuosoRef.current.scrollBy({
            top: 500,
            behavior: 'auto'
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
        
        // Use Virtuoso's built-in method for scrolling to bottom
        this.virtuosoRef.current.autoscrollToBottom();
    }
}

export { LogViewerModel };
