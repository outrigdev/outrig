// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { getDefaultStore, useAtom, useAtomValue } from "jotai";
import {
    BookOpen,
    Box,
    Clock,
    Github,
    Home,
    MessageSquare,
    Moon,
    Settings,
    Sun,
    Twitter,
    X,
    Youtube,
} from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { AppModel } from "../appmodel";
import { cn, formatDuration, formatRelativeTime } from "../util/util";

// AppRunItem component for displaying a single app run item
interface AppRunItemProps {
    appRun: AppRunInfo;
    isSelected: boolean;
}

export const AppRunItem = React.memo<AppRunItemProps>(({ appRun, isSelected }) => {
    const [currentTime, setCurrentTime] = useState(() => Date.now());

    console.log("rendering AppRunItem", appRun.appname, appRun.apprunid, appRun.status);

    // Only update the time for running apps
    useEffect(() => {
        let interval: NodeJS.Timeout | null = null;

        if (appRun.status === "running") {
            interval = setInterval(() => {
                setCurrentTime(Date.now());
            }, 1000);
        }

        // Always return a cleanup function to ensure interval is cleared
        // when status changes or component unmounts
        return () => {
            if (interval) clearInterval(interval);
        };
    }, [appRun.status]);
    return (
        <div
            className={cn(
                "p-2 rounded text-sm cursor-pointer",
                isSelected ? "bg-buttonhover text-primary" : "text-secondary hover:bg-buttonhover hover:text-primary"
            )}
            onClick={() => {
                AppModel.selectAppRun(appRun.apprunid);
                getDefaultStore().set(AppModel.leftNavOpen, false);
            }}
        >
            <div className="flex items-center justify-between">
                {/* Running indicator as part of the flow with visibility */}
                <div
                    className={cn(
                        "w-2 h-2 rounded-full bg-green-500 mr-2",
                        appRun.status === "running" ? "visible" : "invisible"
                    )}
                ></div>
                <div className="flex items-center flex-1">
                    <span className="inline-block w-24">
                        {appRun.status === "running" ? "Running" : formatRelativeTime(appRun.starttime)}
                    </span>
                    <Clock size={12} className="mr-1" />
                    <span>
                        {appRun.status === "running"
                            ? formatDuration(Math.floor((currentTime - appRun.starttime) / 1000))
                            : formatDuration(Math.floor((appRun.lastmodtime - appRun.starttime) / 1000))}
                    </span>
                </div>
                <div className="flex items-center">
                    <span className="text-xs whitespace-nowrap text-muted">({appRun.apprunid.substring(0, 4)})</span>
                </div>
            </div>
        </div>
    );
});

// Add displayName for the memoized component
AppRunItem.displayName = "AppRunItem";

// AppNameGroup component for displaying a group of app runs with the same app name
interface AppNameGroupProps {
    appName: string;
    appRuns: AppRunInfo[];
    selectedAppRunId: string;
}

export const AppNameGroup: React.FC<AppNameGroupProps> = ({ appName, appRuns, selectedAppRunId }) => {
    // Count running apps in this group
    const runningCount = appRuns.filter((run) => run.status === "running").length;

    return (
        <div className="mb-2">
            {/* App Name Header */}
            <div className="flex items-center justify-between px-2 py-1 text-sm font-medium text-primary rounded">
                <div className="flex items-center">
                    <Box size={16} className="mr-1" />
                    <span>{appName}</span>
                </div>
                <div className="text-[10px] text-muted">
                    {runningCount > 0 && (
                        <span className="bg-green-500/10 text-green-500 px-1 py-0.5 rounded">
                            {runningCount} running
                        </span>
                    )}
                </div>
            </div>

            {/* App Runs in this group */}
            <div className="">
                {appRuns.map((appRun) => (
                    <AppRunItem
                        key={appRun.apprunid}
                        appRun={appRun}
                        isSelected={appRun.apprunid === selectedAppRunId}
                    />
                ))}
            </div>
        </div>
    );
};

// AppRunList component for displaying the list of app runs in the left navigation
export const LeftNavAppRunList: React.FC = () => {
    const [isOpen, setIsOpen] = useAtom(AppModel.leftNavOpen);
    const unsortedAppRuns = useAtomValue(AppModel.appRunModel.appRuns);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // Group app runs by app name and sort within groups
    const groupedAppRuns = useMemo(() => {
        // First, get all running app runs
        const runningAppRuns = unsortedAppRuns.filter((run) => run.status === "running");

        // Then get the latest 10 app runs by start time (excluding running ones to avoid duplicates)
        const nonRunningAppRuns = unsortedAppRuns
            .filter((run) => run.status !== "running")
            .sort((a, b) => b.starttime - a.starttime)
            .slice(0, Math.max(0, 10 - runningAppRuns.length));

        // Combine running and non-running app runs
        const filteredAppRuns = [...runningAppRuns, ...nonRunningAppRuns];

        // Check if we're showing all app runs or if some are hidden
        const hasHiddenAppRuns = unsortedAppRuns.length > filteredAppRuns.length;

        // Group app runs by app name
        const groups: Record<string, AppRunInfo[]> = {};

        filteredAppRuns.forEach((appRun) => {
            const appName = appRun.appname || "Unknown";
            if (!groups[appName]) {
                groups[appName] = [];
            }
            groups[appName].push(appRun);
        });

        // Sort app runs within each group: running first, then by start time (newest first)
        Object.keys(groups).forEach((appName) => {
            groups[appName].sort((a, b) => {
                // First sort by status (running at the top)
                if (a.status === "running" && b.status !== "running") return -1;
                if (a.status !== "running" && b.status === "running") return 1;

                // Then sort by start time (newest first)
                return b.starttime - a.starttime;
            });
        });

        // Sort the app names:
        // 1. Groups with running apps first
        // 2. Then by most recent run (using the most recent run in each group)
        // 3. Break ties with app name
        const sortedAppNames = Object.keys(groups).sort((a, b) => {
            const aHasRunning = groups[a].some((run) => run.status === "running");
            const bHasRunning = groups[b].some((run) => run.status === "running");

            // Groups with running apps first
            if (aHasRunning && !bHasRunning) return -1;
            if (!aHasRunning && bHasRunning) return 1;

            // Find the most recent run in each group
            // We've already sorted runs within each group, so the first one is the most recent
            const aMostRecentRun = groups[a][0];
            const bMostRecentRun = groups[b][0];

            // Compare by most recent run time (using the max of start time and last mod time)
            const aLatestTime = Math.max(aMostRecentRun.starttime, aMostRecentRun.lastmodtime);
            const bLatestTime = Math.max(bMostRecentRun.starttime, bMostRecentRun.lastmodtime);

            // Sort by most recent first
            if (aLatestTime > bLatestTime) return -1;
            if (aLatestTime < bLatestTime) return 1;

            // Break ties with app name
            return a.localeCompare(b);
        });

        return { groups, sortedAppNames, hasHiddenAppRuns };
    }, [unsortedAppRuns]);

    return (
        <>
            <div className="px-4 pt-2 pb-1 flex items-center justify-between">
                <span className="text-[10px] font-bold text-secondary uppercase">Recent App Runs</span>
                {/* Show All link if there are hidden app runs */}
                {groupedAppRuns.hasHiddenAppRuns && (
                    <button
                        className="text-secondary hover:text-primary text-[10px] cursor-pointer"
                        onClick={() => {
                            // Navigate to homepage
                            AppModel.navToHomepage();
                            setIsOpen(false);
                        }}
                    >
                        Show All
                    </button>
                )}
            </div>

            {/* App Runs List (Scrollable) */}
            <div className="flex-1 overflow-y-auto">
                {unsortedAppRuns.length === 0 ? (
                    <div className="px-4 py-2 text-secondary text-sm">No app runs found</div>
                ) : (
                    <div className="pl-3 pr-2">
                        {groupedAppRuns.sortedAppNames.map((appName) => (
                            <AppNameGroup
                                key={appName}
                                appName={appName}
                                appRuns={groupedAppRuns.groups[appName]}
                                selectedAppRunId={selectedAppRunId}
                            />
                        ))}
                    </div>
                )}
            </div>
        </>
    );
};

// Theme toggle component
const ThemeToggle: React.FC = () => {
    const darkMode = useAtomValue(AppModel.darkMode);

    const handleToggle = () => {
        AppModel.setDarkMode(!darkMode);
    };

    return (
        <button
            onClick={handleToggle}
            className="w-full flex items-center justify-between p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer"
        >
            <div className="flex items-center space-x-2">
                {darkMode ? <Moon size={16} /> : <Sun size={16} />}
                <span>{darkMode ? "Dark Mode" : "Light Mode"}</span>
            </div>
        </button>
    );
};

export const LeftNav: React.FC = () => {
    const [isOpen, setIsOpen] = useAtom(AppModel.leftNavOpen);

    const handleClose = () => {
        setIsOpen(false);
    };

    if (!isOpen) {
        return null;
    }

    return (
        <>
            {/* Overlay */}
            <div className="fixed inset-0 bg-black/20 backdrop-blur-[1px] z-40" onClick={handleClose} />

            {/* Left Navigation */}
            <div className="fixed top-0 left-0 h-full w-64 bg-panel border-r-2 border-border z-50 flex flex-col transition-transform duration-300 ease-in-out translate-x-0">
                {/* Header with close button */}
                <div
                    className="flex items-center justify-between p-4 border-b border-border cursor-pointer"
                    onClick={() => setIsOpen(false)}
                >
                    <div className="flex items-center space-x-2">
                        <img src="/outriglogo.svg" alt="Outrig Logo" className="w-[20px] h-[20px]" />
                        <span className="text-primary font-medium">Outrig</span>
                    </div>
                    <button
                        onClick={(e) => {
                            e.stopPropagation();
                            handleClose();
                        }}
                        className="text-secondary hover:text-primary cursor-pointer"
                    >
                        <X size={18} />
                    </button>
                </div>

                {/* Navigation Links */}
                <div className="flex-1 overflow-hidden flex flex-col">
                    {/* Top Links */}
                    <div className="p-2 border-b border-border">
                        <button
                            className="w-full flex items-center space-x-2 p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer"
                            onClick={() => {
                                // Navigate to homepage
                                AppModel.navToHomepage();
                                setIsOpen(false);
                            }}
                        >
                            <Home size={16} />
                            <span>Home</span>
                        </button>
                    </div>

                    {/* App Runs Section */}
                    <LeftNavAppRunList />

                    {/* Bottom Links */}
                    <div className="mt-auto border-t border-border p-2">
                        <div className="flex flex-col gap-1">
                            <ThemeToggle />
                            <button
                                className="w-full flex items-center space-x-2 p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer"
                                onClick={() => {
                                    AppModel.openSettingsModal();
                                    setIsOpen(false);
                                }}
                            >
                                <Settings size={16} />
                                <span>Settings</span>
                            </button>
                        </div>
                    </div>

                    {/* Social Links */}
                    <div className="flex justify-center space-x-3 p-4 border-t border-border">
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Github size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <BookOpen size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <MessageSquare size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Youtube size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Twitter size={18} />
                        </a>
                    </div>
                </div>
            </div>
        </>
    );
};
