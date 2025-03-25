import { keydownWrapper } from "@/util/keyutil";
import { getDefaultStore, useAtom, useAtomValue } from "jotai";
import { Check } from "lucide-react";
import { useEffect } from "react";
import { AppModel } from "./appmodel";
import { ToastContainer } from "./elements/toast";
import { Tooltip } from "./elements/tooltip";
import { GoRoutines } from "./goroutines/goroutines";
import { HomePage } from "./homepage/homepage";
import { appHandleKeyDown } from "./keymodel";
import { LogViewer } from "./logviewer/logviewer";
import { LeftNav } from "./main/leftnav";
import { RuntimeStats } from "./runtimestats/runtimestats";
import { StatusBar } from "./statusbar";
import { Watches } from "./watches/watches";

// Define display names for tabs
const TAB_DISPLAY_NAMES: Record<string, string> = {
    logs: "Logs",
    goroutines: "GoRoutines",
    watches: "Watches",
    runtimestats: "Runtime Stats",
};

// Component for rendering feature tabs (logs, goroutines, watches)
function FeatureTab() {
    const selectedTab = useAtomValue(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // We should always have an app run ID here since the parent component
    // conditionally renders the HomePage when no app run is selected
    if (selectedTab === "logs") {
        return <LogViewer key={selectedAppRunId} appRunId={selectedAppRunId} />;
    } else if (selectedTab === "goroutines") {
        return <GoRoutines key={selectedAppRunId} appRunId={selectedAppRunId} />;
    } else if (selectedTab === "watches") {
        return <Watches key={selectedAppRunId} appRunId={selectedAppRunId} />;
    } else if (selectedTab === "runtimestats") {
        return <RuntimeStats key={selectedAppRunId} appRunId={selectedAppRunId} />;
    }

    return <div className="w-full h-full flex items-center justify-center text-secondary">Not Implemented</div>;
}

function AppHeader() {
    const [_, setLeftNavOpen] = useAtom(AppModel.leftNavOpen);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(selectedAppRunId || "");
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const allAppRuns = useAtomValue(AppModel.appRunModel.appRuns);

    const handleHeaderClick = () => {
        setLeftNavOpen(true);
    };

    // Determine if this is the latest run for this app name
    const isLatestRun = (currentRun: AppRunInfo): boolean => {
        if (!currentRun) return false;

        // Filter app runs with the same appname
        const sameAppRuns = allAppRuns.filter((run) => run.appname === currentRun.appname);

        // Sort by starttime (newest first)
        const sortedRuns = [...sameAppRuns].sort((a, b) => b.starttime - a.starttime);

        // Check if the current run is the first (latest) one
        return sortedRuns.length > 0 && sortedRuns[0].apprunid === currentRun.apprunid;
    };

    // Get the label to display
    const getRunLabel = (): string => {
        if (!appRunInfo || !selectedAppRunId) return "";

        if (isLatestRun(appRunInfo)) {
            return "(latest)";
        } else {
            // Show first 4 chars of UUID
            return `(${selectedAppRunId.substring(0, 4)})`;
        }
    };

    return (
        <div className="flex items-center cursor-pointer" onClick={handleHeaderClick}>
            <div className="flex items-center space-x-2">
                <img src="/outriglogo.svg" alt="Outrig Logo" className="w-[20px] h-[20px]" />
            </div>
            {appRunInfo && selectedAppRunId && (
                <>
                    <div className="items-center ml-3 mr-1 text-primary text-sm font-medium max-w-[150px] truncate overflow-hidden whitespace-nowrap">
                        {appRunInfo.appname}
                    </div>
                    <div className="text-xs text-secondary relative top-[1px]">{getRunLabel()}</div>
                </>
            )}
        </div>
    );
}

function AutoFollowButton() {
    const autoFollow = useAtomValue(AppModel.autoFollow);

    const handleToggle = () => {
        AppModel.setAutoFollow(!autoFollow);
    };

    return (
        <Tooltip
            content={
                <span>
                    When selected, auto-follow will automatically
                    <br />
                    switch you to the most recent active app run.
                </span>
            }
        >
            <button
                onClick={handleToggle}
                className="flex items-center gap-2 px-3 py-1 transition-colors cursor-pointer border-l-2 border-gray-300 dark:border-gray-600"
            >
                {/* Modern Toggle Switch */}
                <div className="relative">
                    <div 
                        className={`w-7 h-3.5 rounded-full transition-colors ${
                            autoFollow 
                                ? "bg-sky-500/50" 
                                : "bg-gray-300 dark:bg-gray-600"
                        }`}
                    />
                    <div 
                        className={`absolute top-[-1px] left-0 w-4 h-4 rounded-full shadow-sm transform transition-transform ${
                            autoFollow 
                                ? "translate-x-3.5 bg-sky-500" 
                                : "bg-gray-400 dark:bg-gray-500"
                        }`}
                    />
                </div>
                <span
                    className={`text-xs ${
                        autoFollow ? "text-sky-500 font-medium" : "text-gray-500 dark:text-gray-400"
                    }`}
                >
                    follow new runs
                </span>
            </button>
        </Tooltip>
    );
}

function Tab({ name, displayName }: { name: string; displayName: string }) {
    const [selectedTab, setSelectedTab] = useAtom(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    const handleTabClick = () => {
        // We should always have an app run ID here since the parent component
        // conditionally renders the HomePage when no app run is selected
        if (name === "goroutines") {
            AppModel.selectGoRoutinesTab();
        } else if (name === "logs") {
            AppModel.selectLogsTab();
        } else if (name === "appruns") {
            AppModel.selectAppRunsTab();
        } else if (name === "watches") {
            AppModel.selectWatchesTab();
        } else if (name === "runtimestats") {
            AppModel.selectRuntimeStatsTab();
        } else {
            console.log("unknown tab selected", name);
        }
    };

    return (
        <button
            onClick={handleTabClick}
            data-selected={selectedTab === name || undefined}
            className="relative px-4 py-2 text-secondary text-sm data-[selected]:text-primary
                data-[selected]:after:content-[''] data-[selected]:after:absolute 
                data-[selected]:after:left-0 data-[selected]:after:bottom-[-1px] 
                data-[selected]:after:w-full data-[selected]:after:h-[1px] 
                data-[selected]:after:bg-primary
                hover:after:content-[''] hover:after:absolute 
                hover:after:left-0 hover:after:bottom-[-1px] 
                hover:after:w-full hover:after:h-[1px] 
                hover:after:bg-muted
                hover:text-primary transition-colors cursor-pointer"
        >
            {displayName}
        </button>
    );
}

function App() {
    const darkMode = useAtomValue(AppModel.darkMode);
    const selectedTab = useAtomValue(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const [toasts, setToasts] = useAtom(AppModel.toasts);

    useEffect(() => {
        AppModel.applyTheme();

        const staticKeyDownHandler = keydownWrapper(appHandleKeyDown);
        document.addEventListener("keydown", staticKeyDownHandler);
        return () => {
            document.removeEventListener("keydown", staticKeyDownHandler);
        };
    }, []);

    // Track URL changes and send them to the backend
    useEffect(() => {
        // Send the URL when the component mounts or when tab/appRunId changes
        AppModel.sendBrowserTabUrl();

        // Listen for popstate events (browser back/forward buttons)
        const handlePopState = () => {
            AppModel.handlePopState();
        };

        // Listen for hashchange events
        const handleHashChange = () => {
            AppModel.sendBrowserTabUrl();
        };

        // Listen for focus/blur events to update the focused state
        const handleFocus = () => {
            AppModel.sendBrowserTabUrl();
        };

        const handleBlur = () => {
            AppModel.sendBrowserTabUrl();
        };

        window.addEventListener("popstate", handlePopState);
        window.addEventListener("hashchange", handleHashChange);
        window.addEventListener("focus", handleFocus);
        window.addEventListener("blur", handleBlur);

        // Clean up event listeners
        return () => {
            window.removeEventListener("popstate", handlePopState);
            window.removeEventListener("hashchange", handleHashChange);
            window.removeEventListener("focus", handleFocus);
            window.removeEventListener("blur", handleBlur);
        };
    }, [selectedAppRunId, selectedTab]); // Re-run when selectedAppRunId or selectedTab changes

    // Handle toast removal
    const handleToastClose = (id: string) => {
        AppModel.removeToast(id);
    };

    // If no app run is selected, show the homepage
    if (!selectedAppRunId) {
        return (
            <>
                <HomePage />
                {/* Toast container */}
                <ToastContainer toasts={toasts} onClose={handleToastClose} />
            </>
        );
    }

    // Otherwise, show the main app UI with tabs
    return (
        <div className="h-screen w-screen flex flex-col bg-panel">
            <LeftNav />
            <nav className="bg-panel pl-4 pr-2 border-b border-border flex justify-between items-center">
                <div className="flex items-center">
                    <AppHeader />
                    <div className="mx-3 h-5 w-[2px] bg-gray-300 dark:bg-gray-600"></div>
                    <div className="flex">
                        {/* All tabs require an app run ID now */}
                        {Object.keys(TAB_DISPLAY_NAMES).map((tabName) => (
                            <Tab key={tabName} name={tabName} displayName={TAB_DISPLAY_NAMES[tabName]} />
                        ))}
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <AutoFollowButton />
                </div>
            </nav>

            {/* Main content */}
            <main className="flex-grow overflow-auto w-full">
                <FeatureTab />
            </main>

            {/* Status bar */}
            <StatusBar />

            {/* Toast container */}
            <ToastContainer toasts={toasts} onClose={handleToastClose} />
        </div>
    );
}

export { App };
