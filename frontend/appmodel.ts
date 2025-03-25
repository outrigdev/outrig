import { atom, Atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppRunModel } from "./apprunlist/apprunlist-model";
import { Toast } from "./elements/toast";
import { DefaultRpcClient } from "./init";
import { RpcApi } from "./rpc/rpcclientapi";

const AUTO_FOLLOW_STORAGE_KEY = "outrig:autoFollow";

// Define URL state type
interface UrlState {
    tab?: string | null;
    appRunId?: string | null;
}

// Create a primitive boolean atom.
class AppModel {
    // UI state
    selectedTab: PrimitiveAtom<string> = atom("logs"); // Default to logs view
    darkMode: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem("theme") === "dark");
    autoFollow: PrimitiveAtom<boolean> = atom<boolean>(sessionStorage.getItem(AUTO_FOLLOW_STORAGE_KEY) !== "false"); // Default to true if not set
    leftNavOpen: PrimitiveAtom<boolean> = atom<boolean>(false); // State for left navigation bar

    // Toast notifications
    toasts: PrimitiveAtom<Toast[]> = atom<Toast[]>([]);

    // App run selection
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");

    // App run model
    appRunModel: AppRunModel;

    // Cache for app run info atoms
    appRunInfoAtomCache: Map<string, Atom<AppRunInfo>> = new Map();

    // Flag to prevent URL updates during initialization
    private _isInitializing: boolean = true;

    constructor() {
        this.appRunModel = new AppRunModel();
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
        if (tabParam && ["logs", "goroutines", "watches", "runtimestats"].includes(tabParam)) {
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
    updateUrl(stateChanges: UrlState, usePushState = true) {
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

        // Use pushState to create a new history entry (for back button support)
        // or replaceState to update the current entry without creating a new one
        if (usePushState) {
            window.history.pushState({}, "", newUrl);
        } else {
            window.history.replaceState({}, "", newUrl);
        }
    }

    // Pending state from URL that needs to be verified
    private _pendingAppRunId: string | null = null;
    private _pendingTab: string | null = null;

    async loadAppRuns() {
        // Let errors propagate to the caller
        await this.appRunModel.loadAppRuns();

        // If we have a pending appRunId from URL, verify it exists and set it
        if (this._pendingAppRunId) {
            const appRuns = getDefaultStore().get(this.appRunModel.appRuns);
            const appRunExists = appRuns.some((run) => run.apprunid === this._pendingAppRunId);

            if (appRunExists) {
                const appRunId = this._pendingAppRunId as string;

                // Set the appRunId
                getDefaultStore().set(this.selectedAppRunId, appRunId);

                // Note: Goroutines are loaded by the GoRoutines component when it mounts
                // No need to preload data here
                // Note: Logs are loaded by the LogViewer component when it mounts
                // Note: We don't load any data if we're on the appruns tab
            } else {
                // If appRunId is invalid, clear the appRunId from URL
                this.updateUrl({ appRunId: null });
            }

            // Clear the pending state
            this._pendingAppRunId = null;
            this._pendingTab = null;
        }
    }

    // loadAppRunGoroutines is now handled by the GoRoutinesModel

    selectAppRun(appRunId: string, tab: string = "logs") {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        // Use pushState to create a new history entry for navigating to an app run
        this.updateUrl({ appRunId: appRunId }, true);

        // Set the selected tab
        getDefaultStore().set(this.selectedTab, tab);
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: tab }, false);
    }

    // Select an app run without changing the current tab
    selectAppRunKeepTab(appRunId: string, isAutoFollowSelection = false) {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        // Use pushState to create a new history entry for navigating to an app run
        this.updateUrl({ appRunId: appRunId }, true);

        // If this is a manual selection (not from auto-follow), check if we should disable auto-follow
        if (!isAutoFollowSelection) {
            this.checkAndDisableAutoFollow(appRunId);
        }
    }

    // Check if the selected app run is not the "best" one, and if so, disable auto-follow
    private checkAndDisableAutoFollow(appRunId: string) {
        const bestAppRun = this.appRunModel.findBestAppRun();
        if (bestAppRun && bestAppRun.apprunid !== appRunId) {
            // The selected app run is not the best one, disable auto-follow
            const autoFollow = getDefaultStore().get(this.autoFollow);
            if (autoFollow) {
                console.log(`[AppModel] Disabling auto-follow because user selected a non-best app run`);
                this.setAutoFollow(false);
                this.showToast(
                    "Auto-Follow Disabled",
                    "Auto-follow was disabled because you selected an older app run.",
                    3000
                );
            }
        }
    }

    // Navigate to the homepage
    navToHomepage() {
        // Clear the selected app run ID
        getDefaultStore().set(this.selectedAppRunId, "");
        // Clear the selected tab (or reset to default)
        getDefaultStore().set(this.selectedTab, "logs");
        // Use pushState to create a new history entry for navigating to the homepage
        this.updateUrl({ appRunId: null, tab: null }, true);
    }

    selectLogsTab() {
        // Note: Logs are loaded by the LogViewer component when it mounts
        getDefaultStore().set(this.selectedTab, "logs");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "logs" }, false);
    }

    // This method is kept for backward compatibility
    selectAppRunsTab() {
        // No longer setting a specific tab, just navigating to homepage
        this.navToHomepage();
    }

    selectGoRoutinesTab() {
        // Note: Goroutines are loaded by the GoRoutines component when it mounts
        getDefaultStore().set(this.selectedTab, "goroutines");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "goroutines" }, false);
    }

    selectWatchesTab() {
        // Note: Watches are loaded by the Watches component when it mounts
        getDefaultStore().set(this.selectedTab, "watches");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "watches" }, false);
    }

    selectRuntimeStatsTab() {
        // Note: Runtime Stats are loaded by the RuntimeStats component when it mounts
        getDefaultStore().set(this.selectedTab, "runtimestats");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "runtimestats" }, false);
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

    setAutoFollow(update: boolean): void {
        sessionStorage.setItem(AUTO_FOLLOW_STORAGE_KEY, update.toString());
        getDefaultStore().set(this.autoFollow, update);

        // Send updated browser tab info to the backend
        this.sendBrowserTabUrl();
    }

    getAppRunInfoAtom(appRunId: string): Atom<AppRunInfo> {
        appRunId = appRunId || "";
        if (!this.appRunInfoAtomCache.has(appRunId)) {
            const appRunInfoAtom = atom((get) => {
                if (appRunId === "") {
                    return null;
                }
                const appRuns = get(this.appRunModel.appRuns);
                return appRuns.find((run) => run.apprunid === appRunId);
            });
            this.appRunInfoAtomCache.set(appRunId, appRunInfoAtom);
        }
        return this.appRunInfoAtomCache.get(appRunId)!;
    }

    // Toast management
    showToast(title: string, message: string, timeout?: number): string {
        const id = Date.now().toString();
        const toast: Toast = { id, title, message, timeout };

        const currentToasts = getDefaultStore().get(this.toasts);
        getDefaultStore().set(this.toasts, [...currentToasts, toast]);

        return id;
    }

    removeToast(id: string) {
        const currentToasts = getDefaultStore().get(this.toasts);
        getDefaultStore().set(
            this.toasts,
            currentToasts.filter((toast) => toast.id !== id)
        );
    }

    // Send the current browser tab URL to the backend
    sendBrowserTabUrl() {
        if (!DefaultRpcClient) return;

        const currentUrl = window.location.href;
        const selectedAppRunId = getDefaultStore().get(this.selectedAppRunId);
        const autoFollow = getDefaultStore().get(this.autoFollow);

        // Send the URL, app run ID, focus state, and autofollow state to the backend
        RpcApi.UpdateBrowserTabUrlCommand(DefaultRpcClient, {
            url: currentUrl,
            apprunid: selectedAppRunId || "",
            focused: document.hasFocus(),
            autofollow: autoFollow,
        }).catch((err: Error) => {
            console.error("Failed to send URL to backend:", err);
        });
    }
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
