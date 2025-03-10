// AppModel.ts
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { RpcClient } from "./rpc/rpc";
import { RpcApi } from "./rpc/rpcclientapi";

// Define URL state type
interface UrlState {
    tab?: string | null;
    appRunId?: string | null;
}

// Create a primitive boolean atom.
class AppModel {
    // UI state
    selectedTab: PrimitiveAtom<string> = atom("appruns"); // Default to app runs list view
    darkMode: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem("theme") === "dark");

    // Status metrics
    numGoRoutines: PrimitiveAtom<number> = atom<number>(24);
    numLogLines: PrimitiveAtom<number> = atom<number>(1083);
    appStatus: PrimitiveAtom<"connected" | "disconnected" | "paused"> = atom<"connected" | "disconnected" | "paused">(
        "connected"
    );

    // App runs data
    appRuns: PrimitiveAtom<AppRunInfo[]> = atom<AppRunInfo[]>([]);
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");
    appRunLogs: PrimitiveAtom<LogLine[]> = atom<LogLine[]>([]);
    appRunGoroutines: PrimitiveAtom<GoroutineData[]> = atom<GoroutineData[]>([]);
    isLoadingGoroutines: PrimitiveAtom<boolean> = atom<boolean>(false);

    // Flag to prevent URL updates during initialization
    private _isInitializing: boolean = true;

    // RPC client
    rpcClient: RpcClient | null = null;

    constructor() {
        this.applyTheme();
        this.initFromUrl();
        // Mark initialization as complete
        this._isInitializing = false;
    }

    // Initialize state from URL parameters
    initFromUrl() {
        const params = new URLSearchParams(window.location.search);
        const tabParam = params.get("tab");
        const appRunIdParam = params.get("appRunId");

        // Set the selected tab if it's valid
        if (tabParam && ["appruns", "logs", "goroutines"].includes(tabParam)) {
            getDefaultStore().set(this.selectedTab, tabParam);
        }

        // Store the appRunId from URL to be set after we verify it exists
        if (appRunIdParam) {
            this._pendingAppRunId = appRunIdParam;
            
            // Also store the tab we're on, so we can load the right data
            if (tabParam === "logs" || tabParam === "goroutines") {
                this._pendingTab = tabParam;
            }
        }
    }

    // Get current URL state
    getUrlState(): UrlState {
        return {
            tab: getDefaultStore().get(this.selectedTab),
            appRunId: getDefaultStore().get(this.selectedAppRunId),
        };
    }

    // Update URL with provided state changes
    updateUrl(stateChanges: UrlState) {
        // Skip URL updates during initialization
        if (this._isInitializing) return;

        const params = new URLSearchParams(window.location.search);

        // Process each property in the state changes
        Object.entries(stateChanges).forEach(([key, value]) => {
            if (value === null || value === "") {
                // Remove parameter if value is null or empty string
                params.delete(key);
            } else if (value !== undefined) {
                // Update parameter if value is provided and not empty
                params.set(key, value);
            }
        });

        // Update URL without reloading the page
        const newUrl = `${window.location.pathname}${params.toString() ? "?" + params.toString() : ""}`;
        window.history.replaceState({}, "", newUrl);
    }

    // Pending state from URL that needs to be verified
    private _pendingAppRunId: string | null = null;
    private _pendingTab: string | null = null;

    setRpcClient(client: RpcClient) {
        this.rpcClient = client;
    }

    async loadAppRuns() {
        if (!this.rpcClient) {
            return;
        }

        try {
            const result = await RpcApi.GetAppRunsCommand(this.rpcClient);
            getDefaultStore().set(this.appRuns, result.appruns);

            // If we have a pending appRunId from URL, verify it exists and set it
            if (this._pendingAppRunId) {
                const appRunExists = result.appruns.some((run) => run.apprunid === this._pendingAppRunId);

                if (appRunExists) {
                    const appRunId = this._pendingAppRunId as string;
                    
                    // Set the appRunId
                    getDefaultStore().set(this.selectedAppRunId, appRunId);
                    
                    // Load the appropriate data based on the tab
                    if (this._pendingTab === "goroutines") {
                        this.loadAppRunGoroutines(appRunId);
                    } else if (this._pendingTab === "logs") {
                        // Load logs only if we're on the logs tab
                        this.loadAppRunLogs(appRunId);
                    }
                    // Note: We don't load any data if we're on the appruns tab
                } else {
                    // If appRunId is invalid, switch to appruns tab and remove appRunId from URL
                    getDefaultStore().set(this.selectedTab, "appruns");
                    this.updateUrl({ tab: "appruns", appRunId: null });
                }

                // Clear the pending state
                this._pendingAppRunId = null;
                this._pendingTab = null;
            }
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

    async loadAppRunGoroutines(appRunId: string) {
        if (!this.rpcClient) return;

        try {
            getDefaultStore().set(this.isLoadingGoroutines, true);
            const result = await RpcApi.GetAppRunGoroutinesCommand(this.rpcClient, { apprunid: appRunId });
            getDefaultStore().set(this.appRunGoroutines, result.goroutines);
            getDefaultStore().set(this.selectedAppRunId, appRunId);
        } catch (error) {
            console.error(`Failed to load goroutines for app run ${appRunId}:`, error);
        } finally {
            getDefaultStore().set(this.isLoadingGoroutines, false);
        }
    }

    selectAppRun(appRunId: string) {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        this.loadAppRunLogs(appRunId);
        getDefaultStore().set(this.selectedTab, "logs");
        this.updateUrl({ tab: "logs", appRunId });
    }

    selectGoroutinesTab() {
        const appRunId = getDefaultStore().get(this.selectedAppRunId);
        if (appRunId) {
            this.loadAppRunGoroutines(appRunId);
        }
        getDefaultStore().set(this.selectedTab, "goroutines");
        this.updateUrl({ tab: "goroutines" });
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
