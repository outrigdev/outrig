// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { SettingsButton } from "@/elements/settingsbutton";
import { Tooltip } from "@/elements/tooltip";
import { GoRoutines } from "@/goroutines/goroutines";
import { LogViewer } from "@/logviewer/logviewer";
import { LeftNav } from "@/main/leftnav";
import { RuntimeStats } from "@/runtimestats/runtimestats";
import { Watches } from "@/watches/watches";
import { useAtom, useAtomValue } from "jotai";
import { ChevronRight } from "lucide-react";
import { StatusBar } from "./statusbar";

const TAB_DISPLAY_NAMES: Record<string, string> = {
    logs: "Logs",
    goroutines: "GoRoutines",
    watches: "Watches",
    runtimestats: "Runtime Stats",
};

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
    const [isLeftNavOpen, setLeftNavOpen] = useAtom(AppModel.leftNavOpen);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(selectedAppRunId || "");
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const allAppRuns = useAtomValue(AppModel.appRunModel.appRuns);

    const handleHeaderClick = () => {
        AppModel.setLeftNavOpen(!isLeftNavOpen); // Toggle the left nav
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
        <div
            className="flex items-center cursor-pointer rounded-full ml-1 px-3 py-1 transition 
						   bg-gray-200 hover:bg-gray-300 
						   dark:bg-gray-700 hover:dark:bg-gray-600"
            onClick={handleHeaderClick}
        >
            {!isLeftNavOpen && (
                <div
                    className="flex items-center justify-center mr-2 rounded-r cursor-pointer"
                    onClick={() => {
                        // Navigate to homepage
                        AppModel.navToHomepage();
                        AppModel.setLeftNavOpen(false);
                    }}
                >
                    <img src="/outriglogo.svg" alt="Outrig Logo" className="w-4.5 h-4.5" />
                </div>
            )}
            {/* App name */}
            {appRunInfo && selectedAppRunId && (
                <>
                    <span className="text-sm font-medium text-primary truncate max-w-[120px]">
                        {appRunInfo.appname}
                    </span>
                    <span className="ml-1 text-xs text-secondary">{getRunLabel()}</span>
                    {!isLeftNavOpen && <ChevronRight className="ml-1 w-4 h-4 text-secondary" />}
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
                    switch you to the most recent active app run with the same app name.
                </span>
            }
        >
            <button
                onClick={handleToggle}
                className="flex items-center gap-2 px-3 py-1 transition-colors cursor-pointer border-l-2 border-border"
            >
                <div className="relative">
                    <div
                        className={`w-7 h-3.5 rounded-full transition-colors ${
                            autoFollow ? "bg-accent/50" : "bg-secondary/50"
                        }`}
                    />
                    <div
                        className={`absolute top-[-1px] left-0 w-4 h-4 rounded-full shadow-sm transform transition-transform ${
                            autoFollow ? "translate-x-3.5 bg-accent" : "bg-secondary"
                        }`}
                    />
                </div>
                <span className={`text-xs ${autoFollow ? "text-accent font-medium" : "text-secondary"}`}>
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

export function MainApp() {
    const isLeftNavOpen = useAtomValue(AppModel.leftNavOpen);

    return (
        <div className="flex h-full w-full">
            {/* Left Navigation */}
            <LeftNav />

            {/* Main Content */}
            <div className="flex flex-col flex-grow overflow-hidden">
                <nav className="bg-panel pr-2 border-b border-border flex justify-between items-stretch h-10">
                    <div className="flex items-center">
                        <AppHeader />
                        <div className="flex ml-2">
                            {Object.keys(TAB_DISPLAY_NAMES).map((tabName) => (
                                <Tab key={tabName} name={tabName} displayName={TAB_DISPLAY_NAMES[tabName]} />
                            ))}
                        </div>
                    </div>
                    <div className="flex items-center">
                        <AutoFollowButton />
                        <div className="mx-3 h-5 w-[2px] bg-gray-300 dark:bg-gray-600"></div>
                        <SettingsButton onClick={() => AppModel.openSettingsModal()} />
                    </div>
                </nav>

                <main className="flex-grow overflow-auto w-full">
                    <FeatureTab />
                </main>

                <StatusBar />
            </div>
        </div>
    );
}

export { AppHeader, AutoFollowButton, FeatureTab, Tab };
