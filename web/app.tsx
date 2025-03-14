import { keydownWrapper } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import { Moon, Sun } from "lucide-react";
import { useEffect } from "react";
import { AppModel } from "./appmodel";
import { AppRunList } from "./apprunlist/apprunlist";
import { GoRoutines } from "./goroutines/goroutines";
import { DefaultRpcClient } from "./init";
import { appHandleKeyDown } from "./keymodel";
import { LogViewer } from "./logviewer/logviewer";
import { RpcApi } from "./rpc/rpcclientapi";
import { StatusBar } from "./statusbar";
import { Watches } from "./watches/watches";

// Define tabs that require an app run ID to be selected
// Add new tabs that require an app run ID to this array
const TABS_REQUIRING_APP_RUN_ID = ["logs", "goroutines", "watches"];

// Define display names for tabs
const TAB_DISPLAY_NAMES: Record<string, string> = {
    appruns: "App Runs",
    logs: "Logs",
    goroutines: "GoRoutines",
    watches: "Watches",
};

// Component for rendering the AppRunList
function AppRunsTab() {
    return <AppRunList />;
}

// Component for rendering feature tabs (logs, goroutines, watches)
function FeatureTab() {
    const selectedTab = useAtomValue(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // Return null if no app run is selected
    if (!selectedAppRunId) {
        return null;
    }

    if (selectedTab === "logs") {
        return <LogViewer key={selectedAppRunId} appRunId={selectedAppRunId} />;
    } else if (selectedTab === "goroutines") {
        return <GoRoutines key={selectedAppRunId} appRunId={selectedAppRunId} />;
    } else if (selectedTab === "watches") {
        return <Watches key={selectedAppRunId} appRunId={selectedAppRunId} />;
    }

    return <div className="w-full h-full flex items-center justify-center text-secondary">Not Implemented</div>;
}

function AppLogo() {
    return (
        <div className="flex items-center space-x-2">
            <img src="/outriglogo.svg" alt="Outrig Logo" className="h-5 w-[20px] h-[20px]" />
        </div>
    );
}

function Tab({ name, displayName }: { name: string; displayName: string }) {
    const [selectedTab, setSelectedTab] = useAtom(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    const handleTabClick = () => {
        // If trying to navigate to a tab requiring an app run ID but no app run is selected,
        // don't change the tab (the tabs won't be visible anyway due to conditional rendering)
        if (TABS_REQUIRING_APP_RUN_ID.includes(name) && !selectedAppRunId) {
            return;
        }
        if (name === "goroutines") {
            AppModel.selectGoRoutinesTab();
        } else if (name == "logs") {
            AppModel.selectLogsTab();
        } else if (name == "appruns") {
            AppModel.selectAppRunsTab();
        } else if (name == "watches") {
            AppModel.selectWatchesTab();
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
        // Function to send the current URL to the backend
        const sendUrlToBackend = () => {
            if (!DefaultRpcClient) return;

            const currentUrl = window.location.href;

            // Send the URL and app run ID to the backend
            RpcApi.UpdateBrowserTabUrlCommand(DefaultRpcClient, {
                url: currentUrl,
                apprunid: selectedAppRunId || "",
            }).catch((err: Error) => {
                console.error("Failed to send URL to backend:", err);
            });
        };

        // Send the URL when the component mounts
        sendUrlToBackend();

        // Listen for popstate events (browser back/forward buttons)
        const handlePopState = () => {
            sendUrlToBackend();
        };

        // Listen for hashchange events
        const handleHashChange = () => {
            sendUrlToBackend();
        };

        window.addEventListener("popstate", handlePopState);
        window.addEventListener("hashchange", handleHashChange);

        // Clean up event listeners
        return () => {
            window.removeEventListener("popstate", handlePopState);
            window.removeEventListener("hashchange", handleHashChange);
        };
    }, [selectedAppRunId]); // Re-run when selectedAppRunId changes

    return (
        <div className="h-screen w-screen flex flex-col bg-panel">
            <nav className="bg-panel pl-4 pr-2 border-b border-border flex justify-between items-center">
                <div className="flex items-center">
                    <AppLogo />
                    <div className="ml-3 flex">
                        <Tab name="appruns" displayName={TAB_DISPLAY_NAMES.appruns} />
                        {selectedAppRunId && (
                            <>
                                {TABS_REQUIRING_APP_RUN_ID.map((tabName) => (
                                    <Tab
                                        key={tabName}
                                        name={tabName}
                                        displayName={TAB_DISPLAY_NAMES[tabName] || tabName}
                                    />
                                ))}
                            </>
                        )}
                    </div>
                </div>
                <button
                    onClick={() => AppModel.setDarkMode(!darkMode)}
                    className="p-1 border-none text-secondary hover:bg-buttonhover transition-colors cursor-pointer"
                >
                    {darkMode ? <Moon size={16} /> : <Sun size={16} />}
                </button>
            </nav>

            {/* Main content */}
            <main
                className="flex-grow overflow-auto w-full"
                style={{ display: selectedTab === "appruns" ? "block" : "none" }}
            >
                <AppRunsTab />
            </main>
            <main
                className="flex-grow overflow-auto w-full"
                style={{ display: selectedTab === "appruns" ? "none" : "block" }}
            >
                <FeatureTab />
            </main>

            {/* Status bar */}
            <StatusBar />
        </div>
    );
}

export { App };
