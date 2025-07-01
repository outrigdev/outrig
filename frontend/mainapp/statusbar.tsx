// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AppRunListModel } from "@/apprunlist/apprunlist-model";
import { Tooltip } from "@/elements/tooltip";
import { useAtomValue, useSetAtom } from "jotai";
import { Box, CircleDot, Eye, List, PauseCircle, Wifi, WifiOff } from "lucide-react";
import { useMemo } from "react";

const OutrigVersion = "v" + import.meta.env.PACKAGE_VERSION;

function ConnectionStatus({ status }: { status: string }) {
    let icon;
    let displayName;

    switch (status) {
        case "running":
            icon = <Wifi size={12} />;
            displayName = "Running";
            break;
        case "disconnected":
            icon = <WifiOff size={12} />;
            displayName = "Disconnected";
            break;
        case "paused":
            icon = <PauseCircle size={12} />;
            displayName = "Paused";
            break;
        case "done":
            icon = <Box size={12} />;
            displayName = "Done";
            break;
        default:
            icon = <Wifi size={12} />;
            displayName = status || "Unknown";
    }

    return (
        <div className="flex items-center space-x-1">
            {icon}
            <span>{displayName}</span>
        </div>
    );
}

export function StatusBar() {
    const appRuns = useAtomValue(AppRunListModel.appRuns);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const setSelectedTab = useSetAtom(AppModel.selectedTab);

    // Find the selected app run
    const selectedAppRun = useMemo(() => {
        return appRuns.find((run: AppRunInfo) => run.apprunid === selectedAppRunId);
    }, [appRuns, selectedAppRunId]);

    // Count running app runs
    const runningAppRunsCount = useMemo(() => {
        return appRuns.filter((run: AppRunInfo) => run.status === "running").length;
    }, [appRuns]);

    // Determine which goroutine count to display
    const goroutineCount = useMemo(() => {
        if (!selectedAppRun) return 0;

        // Show active goroutines minus outrig goroutines
        return selectedAppRun.numactivegoroutines - selectedAppRun.numoutriggoroutines;
    }, [selectedAppRun]);

    // Determine the tooltip text for goroutines
    const goroutineTooltip = useMemo(() => {
        if (!selectedAppRun) return "";

        const goroutineCount = selectedAppRun.numactivegoroutines - selectedAppRun.numoutriggoroutines;
        return `${goroutineCount} GoRoutines`;
    }, [selectedAppRun]);

    return (
        <div className="h-6 bg-panel border-t border-border flex items-center justify-between px-2 text-xs text-secondary shrink-0">
            <div className="flex items-center space-x-4">
                <Tooltip content="Running in Development Mode" placement="bottom">
                    <div className="flex items-center">
                        {/* Custom styling for the DEV badge in the status bar */}
                        <span className="px-1.5 py-0 text-[10px] font-bold rounded-md bg-accentbg text-secondary mr-2 leading-[16px]">
                            {AppModel.isDev ? "DEV" : ""} {OutrigVersion}
                        </span>
                    </div>
                </Tooltip>
                {selectedAppRun ? (
                    <>
                        <div className="flex items-center space-x-1">
                            <Box size={12} />
                            <span>{selectedAppRun.appname}</span>
                            <span className="text-muted">({selectedAppRun.apprunid.substring(0, 4)})</span>
                        </div>
                        <ConnectionStatus status={selectedAppRun.status} />
                    </>
                ) : (
                    <div className="flex items-center space-x-1">
                        <span>No App Run Selected</span>
                        <span className="text-muted">({runningAppRunsCount} running)</span>
                    </div>
                )}
            </div>
            {selectedAppRun && (
                <div className="flex items-center space-x-4">
                    <Tooltip content={`${selectedAppRun.numlogs} Log Lines`} placement="bottom">
                        <div
                            className="flex items-center space-x-1 cursor-pointer"
                            onClick={() => setSelectedTab("logs")}
                        >
                            <List size={12} />
                            <span>{selectedAppRun.numlogs}</span>
                        </div>
                    </Tooltip>
                    <Tooltip content={goroutineTooltip} placement="bottom">
                        <div
                            className="flex items-center space-x-1 cursor-pointer"
                            onClick={() => setSelectedTab("goroutines")}
                        >
                            <CircleDot size={12} />
                            <span>{goroutineCount}</span>
                        </div>
                    </Tooltip>
                    {selectedAppRun.numactivewatches > 0 && (
                        <Tooltip
                            content={`${selectedAppRun.numactivewatches} Active ${selectedAppRun.numactivewatches === 1 ? "Watch" : "Watches"}`}
                            placement="bottom"
                        >
                            <div
                                className="flex items-center space-x-1 cursor-pointer"
                                onClick={() => setSelectedTab("watches")}
                            >
                                <Eye size={12} />
                                <span>{selectedAppRun.numactivewatches}</span>
                            </div>
                        </Tooltip>
                    )}
                </div>
            )}
        </div>
    );
}
