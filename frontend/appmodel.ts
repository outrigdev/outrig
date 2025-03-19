import { atom, Atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppRunModel } from "./apprunlist/apprunlist-model";
import { Toast } from "./elements/toast";

const AUTO_FOLLOW_STORAGE_KEY = "outrig:autoFollow";

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
    autoFollow: PrimitiveAtom<boolean> = atom<boolean>(sessionStorage.getItem(AUTO_FOLLOW_STORAGE_KEY) !== "false"); // Default to true if not set

    // Toast notifications
    toasts: PrimitiveAtom<Toast[]> = atom<Toast[]>([]);

    // App run selection
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");

    // App run model
    appRunModel: AppRunModel;

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
        if (tabParam && ["appruns", "logs", "goroutines", "watches", "runtimestats"].includes(tabParam)) {
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
                // If appRunId is invalid, switch to appruns tab and remove appRunId from URL
                getDefaultStore().set(this.selectedTab, "appruns");
                this.updateUrl({ tab: "appruns", appRunId: null });
            }

            // Clear the pending state
            this._pendingAppRunId = null;
            this._pendingTab = null;
        }
    }

    // loadAppRunGoroutines is now handled by the GoRoutinesModel

    selectAppRun(appRunId: string, isAutoFollowSelection = false) {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        this.updateUrl({ appRunId: appRunId });
        this.selectLogsTab();

        // If this is a manual selection (not from auto-follow), check if we should disable auto-follow
        if (!isAutoFollowSelection) {
            this.checkAndDisableAutoFollow(appRunId);
        }
    }

    // Select an app run without changing the current tab
    selectAppRunKeepTab(appRunId: string, isAutoFollowSelection = false) {
        getDefaultStore().set(this.selectedAppRunId, appRunId);
        this.updateUrl({ appRunId: appRunId });

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

    // Clear the selected app run and go to app runs tab
    clearAppRunSelection() {
        getDefaultStore().set(this.selectedAppRunId, "");
        this.updateUrl({ appRunId: null });
        this.selectAppRunsTab();
    }

    selectLogsTab() {
        // Note: Logs are loaded by the LogViewer component when it mounts
        getDefaultStore().set(this.selectedTab, "logs");
        this.updateUrl({ tab: "logs" });
    }

    selectAppRunsTab() {
        getDefaultStore().set(this.selectedTab, "appruns");
        this.updateUrl({ tab: "appruns" });
    }

    selectGoRoutinesTab() {
        // Note: Goroutines are loaded by the GoRoutines component when it mounts
        getDefaultStore().set(this.selectedTab, "goroutines");
        this.updateUrl({ tab: "goroutines" });
    }

    selectWatchesTab() {
        // Note: Watches are loaded by the Watches component when it mounts
        getDefaultStore().set(this.selectedTab, "watches");
        this.updateUrl({ tab: "watches" });
    }

    selectRuntimeStatsTab() {
        // Note: Runtime Stats are loaded by the RuntimeStats component when it mounts
        getDefaultStore().set(this.selectedTab, "runtimestats");
        this.updateUrl({ tab: "runtimestats" });
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
    }

    getAppRunInfoAtom(appRunId: string): Atom<AppRunInfo> {
        return atom((get) => {
            const appRuns = get(this.appRunModel.appRuns);
            return appRuns.find((run) => run.apprunid === appRunId);
        });
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
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
