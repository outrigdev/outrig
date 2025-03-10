import { useAtom, useAtomValue } from "jotai";
import { Moon, Sun } from "lucide-react";
import { useEffect } from "react";
import { AppModel } from "./appmodel";
import { AppRunList } from "./apprunlist/apprunlist";
import { GoRoutines } from "./goroutines/goroutines";
import { DefaultRpcClient } from "./init";
import { LogViewer } from "./logviewer/logviewer";
import { StatusBar } from "./statusbar";

// Define tabs that require an app run ID to be selected
// Add new tabs that require an app run ID to this array
const TABS_REQUIRING_APP_RUN_ID = ["logs", "goroutines"];

function MainTab() {
    const selectedTab = useAtomValue(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // If a tab requiring an app run ID is selected but no app run is selected, show app runs list
    if (TABS_REQUIRING_APP_RUN_ID.includes(selectedTab) && !selectedAppRunId) {
        return <AppRunList />;
    }

    if (selectedTab === "logs") {
        return <LogViewer />;
    } else if (selectedTab === "appruns") {
        return <AppRunList />;
    } else if (selectedTab === "goroutines") {
        return <GoRoutines />;
    }

    return <div className="w-full h-full flex items-center justify-center text-secondary">Not Implemented</div>;
}

function AppLogo() {
    return (
        <div className="flex items-center space-x-2">
            <img src="/outriglogo.svg" alt="Outrig Logo" className="h-5" />
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
            AppModel.selectGoroutinesTab();
        } else {
            setSelectedTab(name);
            // Update URL when tab changes
            AppModel.updateUrl({ tab: name });
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

        // Set the default RPC client
        AppModel.setRpcClient(DefaultRpcClient);

        // Load app runs after setting the RPC client
        AppModel.loadAppRuns();
    }, []);

    // We no longer need this effect as URL updates are handled directly in the AppModel methods

    return (
        <div className="h-screen w-screen flex flex-col bg-panel">
            <nav className="bg-panel px-4 border-b border-border flex justify-between items-center">
                <div className="flex items-center">
                    <AppLogo />
                    <div className="ml-3 flex">
                        <Tab name="appruns" displayName="App Runs" />
                        {selectedAppRunId && (
                            <>
                                {TABS_REQUIRING_APP_RUN_ID.map((tabName) => {
                                    const displayNames: Record<string, string> = {
                                        logs: "Logs",
                                        goroutines: "GoRoutines",
                                    };
                                    return (
                                        <Tab
                                            key={tabName}
                                            name={tabName}
                                            displayName={displayNames[tabName] || tabName}
                                        />
                                    );
                                })}
                            </>
                        )}
                    </div>
                </div>
                <button
                    onClick={() => AppModel.setDarkMode(!darkMode)}
                    className="p-1.5 border border-border rounded-md text-primary hover:bg-buttonhover transition-colors cursor-pointer"
                >
                    {darkMode ? <Moon size={14} /> : <Sun size={14} />}
                </button>
            </nav>

            {/* Main content */}
            <main className="flex-grow overflow-auto w-full">
                <MainTab />
            </main>

            {/* Status bar */}
            <StatusBar />
        </div>
    );
}

export { App };
