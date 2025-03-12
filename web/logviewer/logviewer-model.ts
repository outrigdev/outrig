import { DefaultRpcClient } from "@/init";
import { PromiseQueue } from "@/util/promisequeue";
import { Atom, atom, getDefaultStore, Getter, PrimitiveAtom } from "jotai";
import { VariableSizeList as List } from "react-window";
import { RpcApi } from "../rpc/rpcclientapi";

const PAGESIZE = 100;

type LogCacheEntry = {
    status: "init" | "loading" | "loaded";
    lines: LogLine[];
};

export type SearchType = "exact" | "exactcase" | "regexp" | "fzf";

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    searchTerm: PrimitiveAtom<string> = atom("");
    searchType: PrimitiveAtom<SearchType> = atom<SearchType>("exact");
    isRefreshing: PrimitiveAtom<boolean> = atom(false);
    isLoading: PrimitiveAtom<boolean> = atom(false);
    listRef: React.RefObject<List> = null;

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
            getDefaultStore().set(this.totalItemCount, results.totalcount);
            getDefaultStore().set(this.filteredItemCount, results.filteredcount);
            getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);
            this.logItemCache = [];
            this.setLogCacheEntry(0, "loaded", results.lines);
        } catch (e) {
            this.setLogCacheEntry(0, "loaded", []);
            console.error("Log search error", e);
        } finally {
            clearTimeout(quickSearchTimeoutId);
            getDefaultStore().set(this.isLoading, false);
            const endTime = performance.now();
            console.log("Log search took", endTime - startTime, "ms");
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
        getDefaultStore().set(this.logItemCacheVersion, (version) => version + 1);
        this.logItemCache = [];
        try {
            await this.onSearchTermUpdate(store.get(this.searchTerm), store.get(this.searchType));
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }

    setListRef(ref: React.RefObject<List>) {
        this.listRef = ref;
    }

    pageUp() {
        if (!this.listRef?.current) return;

        // Access scrollOffset using type assertion
        const currentScrollOffset = (this.listRef.current.state as any).scrollOffset || 0;
        const scrollHeight = this.listRef.current.props.height as number;
        this.listRef.current.scrollTo(Math.max(0, currentScrollOffset - scrollHeight));
    }

    pageDown() {
        if (!this.listRef?.current) return;

        // Access scrollOffset using type assertion
        const currentScrollOffset = (this.listRef.current.state as any).scrollOffset || 0;
        const scrollHeight = this.listRef.current.props.height as number;
        this.listRef.current.scrollTo(currentScrollOffset + scrollHeight);
    }

    scrollToTop() {
        if (!this.listRef?.current) return;
        this.listRef.current.scrollTo(0);
    }

    scrollToBottom() {
        if (!this.listRef?.current) return;

        // Get the total height of all items
        const filteredCount = getDefaultStore().get(this.filteredItemCount);
        if (filteredCount <= 0) return;

        // Scroll to a very large number to ensure we reach the bottom
        // react-window will clamp this to the maximum scroll offset
        this.listRef.current.scrollTo(Number.MAX_SAFE_INTEGER);
    }
}

export { LogViewerModel };
