// AppModel.ts
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcClient } from "./rpc/rpc";
import { RpcApi } from "./rpc/rpcclientapi";

// Create a primitive boolean atom.
class AppModel {
    // UI state
    selectedTab: PrimitiveAtom<string> = atom("appruns"); // Default to app runs list view
    darkMode: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem("theme") === "dark");
    
    // Status metrics
    numGoRoutines: PrimitiveAtom<number> = atom<number>(24);
    numLogLines: PrimitiveAtom<number> = atom<number>(1083);
    appStatus: PrimitiveAtom<"connected" | "disconnected" | "paused"> = atom<"connected" | "disconnected" | "paused">("connected");

    // App runs data
    appRuns: PrimitiveAtom<AppRunInfo[]> = atom<AppRunInfo[]>([]);
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");
    appRunLogs: PrimitiveAtom<LogLine[]> = atom<LogLine[]>([]);
    
    // RPC client
    rpcClient: RpcClient | null = null;

    constructor() {
        this.applyTheme();
    }

    setRpcClient(client: RpcClient) {
        this.rpcClient = client;
    }

    async loadAppRuns() {
        if (!this.rpcClient) return;
        
        try {
            const result = await RpcApi.GetAppRunsCommand(this.rpcClient);
            getDefaultStore().set(this.appRuns, result.appruns);
        } catch (error) {
            console.error("Failed to load app runs:", error);
        }
    }

    async loadAppRunLogs(appRunId: string) {
        if (!this.rpcClient) return;
        
        try {
            const result = await RpcApi.GetAppRunLogsCommand(this.rpcClient, { apprunid: appRunId });
            getDefaultStore().set(this.appRunLogs, result.logs);
            getDefaultStore().set(this.selectedAppRunId, appRunId);
        } catch (error) {
            console.error(`Failed to load logs for app run ${appRunId}:`, error);
        }
    }

    selectAppRun(appRunId: string) {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        this.loadAppRunLogs(appRunId);
        getDefaultStore().set(this.selectedTab, "logs");
    }

    applyTheme(): void {
        if (localStorage.getItem("theme") === "dark") {
            document.documentElement.dataset.theme = "dark";
        } else {
            document.documentElement.dataset.theme = "light";
        }
    }

    setDarkMode(update: boolean): void {
        if (update) {
            localStorage.setItem("theme", "dark");
        } else {
            localStorage.setItem("theme", "light");
        }
        this.applyTheme();
        getDefaultStore().set(this.darkMode, update);
    }
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
