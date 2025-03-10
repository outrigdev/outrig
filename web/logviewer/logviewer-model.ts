import { Atom, atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";
import { RpcApi } from "../rpc/rpcclientapi";

class LogViewerModel {
    appRunId: string;
    appRunLogs: PrimitiveAtom<LogLine[]> = atom<LogLine[]>([]);
    searchTerm: PrimitiveAtom<string> = atom("");

    constructor(appRunId: string) {
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

    async loadAppRunLogs() {
        if (!AppModel.rpcClient) return;

        try {
            const result = await RpcApi.GetAppRunLogsCommand(AppModel.rpcClient, { apprunid: this.appRunId });
            getDefaultStore().set(this.appRunLogs, result.logs);
        } catch (error) {
            console.error(`Failed to load logs for app run ${this.appRunId}:`, error);
            getDefaultStore().set(this.appRunLogs, []);
        }
    }
}

export { LogViewerModel };
