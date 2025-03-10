import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";
import { RpcApi } from "../rpc/rpcclientapi";

class LogViewerModel {
    widgetId: string;
    appRunId: string;
    appRunLogs: PrimitiveAtom<LogLine[]> = atom<LogLine[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");
    refreshVersion: PrimitiveAtom<number> = atom(0);
    isRefreshing: PrimitiveAtom<boolean> = atom(false);

    constructor(appRunId: string) {
        this.widgetId = crypto.randomUUID();
        this.appRunId = appRunId;
    }

    filteredLogLines: Atom<LogLine[]> = atom((get) => {
        const search = get(this.searchTerm);
        const logs = get(this.appRunLogs);

        if (!search) {
            return logs;
        }

        return logs.filter((log) => log.msg.toLowerCase().includes(search.toLowerCase()));
    });

    async fetchAppRunLogs() {
        if (!AppModel.rpcClient) return;

        try {
            const result = await RpcApi.GetAppRunLogsCommand(AppModel.rpcClient, { apprunid: this.appRunId });
            return result.logs;
        } catch (error) {
            console.error(`Failed to load logs for app run ${this.appRunId}:`, error);
            return [];
        }
    }

    async loadAppRunLogs(minTime: number = 0) {
        if (!AppModel.rpcClient) return;
        const startTime = new Date().getTime();
        const logs = await this.fetchAppRunLogs();
        if (minTime > 0) {
            const curTime = new Date().getTime();
            if (curTime - startTime < minTime) {
                await new Promise((r) => setTimeout(r, minTime - (curTime - startTime)));
            }
        }
        getDefaultStore().set(this.appRunLogs, logs);
    }

    async refresh() {
        const store = getDefaultStore();

        // If already refreshing, don't do anything
        if (store.get(this.isRefreshing)) {
            return;
        }

        // Set refreshing state to true
        store.set(this.isRefreshing, true);

        // Clear logs immediately
        store.set(this.appRunLogs, []);

        // Increment refresh version
        const currentVersion = store.get(this.refreshVersion);
        store.set(this.refreshVersion, currentVersion + 1);
        try {
            // Load new logs immediately
            await this.loadAppRunLogs(500);
        } finally {
            // Set refreshing state to false
            store.set(this.isRefreshing, false);
        }
    }
}

export { LogViewerModel };
