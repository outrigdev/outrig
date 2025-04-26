// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, Atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppRunModel } from "./apprunlist/apprunlist-model";
import { Toast } from "./elements/toast";
import { emitter } from "./events";
import { DefaultRpcClient } from "./init";
import { RpcApi } from "./rpc/rpcclientapi";
import { sendTabEvent } from "./tevent";
import { isBlank } from "./util/util";

const AutoFollowStorageKey = "outrig:autoFollow";
const ThemeLocalStorageKey = "outrig:theme";
const LeftNavOpenStorageKey = "outrig:leftNavOpen";

// Define URL state type
interface UrlState {
    tab?: string | null;
    appRunId?: string | null;
}

// Create a primitive boolean atom.
class AppModel {
    // Development mode flag
    isDev = import.meta.env.DEV;
    // UI state
    selectedTab: PrimitiveAtom<string> = atom("logs"); // Default to logs view
    darkMode: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem(ThemeLocalStorageKey) !== "light");
    autoFollow: PrimitiveAtom<boolean> = atom<boolean>(sessionStorage.getItem(AutoFollowStorageKey) !== "false"); // Default to true if not set
    leftNavOpen: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem(LeftNavOpenStorageKey) === "true"); // State for left navigation bar
    settingsModalOpen: PrimitiveAtom<boolean> = atom<boolean>(false); // State for settings modal
    newerVersion: PrimitiveAtom<string> = atom(null) as PrimitiveAtom<string>; // Newer version available

    // Toast notifications
    toasts: PrimitiveAtom<Toast[]> = atom<Toast[]>([]);

    // App run selection
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");

    // App run start time
    appRunStartTimeAtom: Atom<number | null> = atom((get) => {
        const appRunId = get(this.selectedAppRunId);
        if (!appRunId) return null;
        const appRunInfo = get(this.getAppRunInfoAtom(appRunId));
        return appRunInfo?.starttime || null;
    });

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
        // Check for updates with a small delay to avoid conflicts with other calls
        setTimeout(() => this.checkForUpdates(), 1000);
        // Mark initialization as complete
        this._isInitializing = false;
    }

    // Check for newer version
    async checkForUpdates() {
        if (!DefaultRpcClient) return;

        try {
            const updateData = await RpcApi.UpdateCheckCommand(DefaultRpcClient);
            if (updateData) {
                if (isBlank(updateData.newerversion)) {
                    getDefaultStore().set(this.newerVersion, null);
                } else {
                    getDefaultStore().set(this.newerVersion, updateData.newerversion);
                }
            }
        } catch (err) {
            console.error("Failed to check for updates:", err);
        }
    }

    // Initialize state from URL parameters
    initFromUrl() {
        const params = new URLSearchParams(window.location.search);
        const tabParam = params.get("tab");
        const appRunIdParam = params.get("appRunId");

        // Set the selected tab if it's valid
        if (tabParam && ["logs", "goroutines", "watches", "runtimestats"].includes(tabParam)) {
            getDefaultStore().set(this.selectedTab, tabParam);
            
            // Send tab event on initial load/refresh
            // We use setTimeout to ensure this happens after RPC client is initialized
            setTimeout(() => {
                if (DefaultRpcClient) {
                    sendTabEvent(tabParam);
                }
            }, 500);
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

        this.checkAndDisableAutoFollow(appRunId);
        // Check for updates when selecting a new app run with a small delay
        setTimeout(() => this.checkForUpdates(), 500);
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
        
        // Check for updates when selecting a new app run with a small delay
        setTimeout(() => this.checkForUpdates(), 500);
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
        // Send tab event
        sendTabEvent("logs");
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
        // Send tab event
        sendTabEvent("goroutines");
    }

    selectWatchesTab() {
        // Note: Watches are loaded by the Watches component when it mounts
        getDefaultStore().set(this.selectedTab, "watches");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "watches" }, false);
        // Send tab event
        sendTabEvent("watches");
    }

    selectRuntimeStatsTab() {
        // Note: Runtime Stats are loaded by the RuntimeStats component when it mounts
        getDefaultStore().set(this.selectedTab, "runtimestats");
        // Use replaceState for tab navigation (no history entry)
        this.updateUrl({ tab: "runtimestats" }, false);
        // Send tab event
        sendTabEvent("runtimestats");
    }

    applyTheme(): void {
        if (localStorage.getItem(ThemeLocalStorageKey) === "light") {
            document.documentElement.dataset.theme = "light";
        } else {
            document.documentElement.dataset.theme = "dark";
        }
    }

    setDarkMode(update: boolean): void {
        if (update) {
            localStorage.setItem(ThemeLocalStorageKey, "dark");
        } else {
            localStorage.setItem(ThemeLocalStorageKey, "light");
        }
        this.applyTheme();
        getDefaultStore().set(this.darkMode, update);
    }

    setAutoFollow(update: boolean): void {
        sessionStorage.setItem(AutoFollowStorageKey, update.toString());
        getDefaultStore().set(this.autoFollow, update);

        // Send updated browser tab info to the backend
        this.sendBrowserTabUrl();
    }

    setLeftNavOpen(update: boolean): void {
        localStorage.setItem(LeftNavOpenStorageKey, update.toString());
        getDefaultStore().set(this.leftNavOpen, update);
    }

    openSettingsModal(): void {
        getDefaultStore().set(this.settingsModalOpen, true);

        // Blur any active element to ensure it doesn't receive input
        if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
        }
    }

    closeSettingsModal(): void {
        getDefaultStore().set(this.settingsModalOpen, false);
        emitter.emit("modalclose");
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

    // Handle browser popstate events (back/forward buttons)
    handlePopState() {
        // Update app state based on URL when navigating with browser back/forward buttons
        const params = new URLSearchParams(window.location.search);
        const tabParam = params.get("tab");
        const appRunIdParam = params.get("appRunId");

        const selectedTab = getDefaultStore().get(this.selectedTab);
        const selectedAppRunId = getDefaultStore().get(this.selectedAppRunId);

        // Update the selected app run ID
        if (appRunIdParam) {
            // Only update if it's different from the current selection
            if (appRunIdParam !== selectedAppRunId) {
                getDefaultStore().set(this.selectedAppRunId, appRunIdParam);
            }
        } else {
            // Clear the selection if there's no app run ID in the URL
            if (selectedAppRunId) {
                getDefaultStore().set(this.selectedAppRunId, "");
            }
        }

        // Update the selected tab
        if (tabParam && ["logs", "goroutines", "watches", "runtimestats"].includes(tabParam)) {
            // Only update if it's different from the current selection
            if (tabParam !== selectedTab) {
                getDefaultStore().set(this.selectedTab, tabParam);
            }
        }

        // Send the updated URL to the backend
        this.sendBrowserTabUrl();
    }
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
