// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { useAtomValue } from "jotai";
import { CircleDot, Clock, ExternalLink, Eye, List } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AppModel } from "../appmodel";
import { Tag } from "../elements/tag";
import { cn, formatDuration, formatRelativeTime } from "../util/util";

interface AppRunStatusTagProps {
    status: string;
}

const AppRunStatusTag: React.FC<AppRunStatusTagProps> = ({ status }) => {
    if (status === "running") {
        return <Tag label="Running" variant="success" isSelected={true} />;
    } else if (status === "done") {
        return <Tag label="Done" variant="secondary" isSelected={true} />;
    } else {
        return <Tag label="Disconnected" variant="secondary" isSelected={true} />;
    }
};

interface AppRunItemProps {
    appRun: AppRunInfo;
    onClick: (appRunId: string) => void;
    isSelected: boolean;
}

const AppRunItem: React.FC<AppRunItemProps> = ({ appRun, onClick, isSelected }) => {
    const [currentTime, setCurrentTime] = useState(() => Date.now());

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

    // Create URL for the app run (for right-click "open in new tab" functionality)
    const appRunUrl = `?tab=logs&appRunId=${appRun.apprunid}`;

    // Calculate duration
    const duration =
        appRun.status === "running"
            ? Math.floor((currentTime - appRun.starttime) / 1000)
            : Math.floor((appRun.lastmodtime - appRun.starttime) / 1000);

    return (
        <div
            className={cn(
                "p-4 hover:bg-buttonhover cursor-pointer block relative group",
                isSelected && "bg-buttonhover border-l-4 border-l-accent"
            )}
            onClick={() => onClick(appRun.apprunid)}
        >
            <div className="flex justify-between items-center">
                <div className="font-medium text-primary flex items-center">
                    {appRun.appname}
                    {appRun.status === "running" && <div className="ml-2 w-2 h-2 rounded-full bg-green-500"></div>}
                    <a
                        href={appRunUrl}
                        className="ml-2 opacity-0 group-hover:opacity-100 transition-opacity text-muted hover:text-primary"
                        title="Open in new tab"
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <ExternalLink size={14} />
                    </a>
                </div>
                <div className="text-xs text-secondary">
                    <AppRunStatusTag status={appRun.status} />
                </div>
            </div>
            <div className="mt-1 text-sm text-secondary">
                {appRun.status === "running"
                    ? "Running"
                    : `${appRun.status === "done" ? "Completed" : "Disconnected"} ${formatRelativeTime(appRun.lastmodtime)}`}
            </div>
            <div className="mt-1 flex items-center space-x-4 text-xs text-muted">
                <a
                    href={`?tab=runtimestats&appRunId=${appRun.apprunid}`}
                    className="flex items-center space-x-1 hover:text-primary hover:underline cursor-pointer"
                    onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        AppModel.selectAppRun(appRun.apprunid, "runtimestats");
                    }}
                >
                    <Clock size={12} />
                    <span>{formatDuration(duration)}</span>
                </a>
                <a
                    href={`?tab=logs&appRunId=${appRun.apprunid}`}
                    className="flex items-center space-x-1 hover:text-primary hover:underline cursor-pointer"
                    onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        AppModel.selectAppRun(appRun.apprunid, "logs");
                    }}
                >
                    <List size={12} />
                    <span>{appRun.numlogs}</span>
                </a>
                <a
                    href={`?tab=goroutines&appRunId=${appRun.apprunid}`}
                    className="flex items-center space-x-1 hover:text-primary hover:underline cursor-pointer"
                    onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        AppModel.selectAppRun(appRun.apprunid, "goroutines");
                    }}
                >
                    <CircleDot size={12} />
                    <span>{appRun.numactivegoroutines - appRun.numoutriggoroutines}</span>
                </a>
                {appRun.numactivewatches > 0 && (
                    <a
                        href={`?tab=watches&appRunId=${appRun.apprunid}`}
                        className="flex items-center space-x-1 hover:text-primary hover:underline cursor-pointer"
                        onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            AppModel.selectAppRun(appRun.apprunid, "watches");
                        }}
                    >
                        <Eye size={12} />
                        <span>{appRun.numactivewatches}</span>
                    </a>
                )}
                <div className="text-muted">({appRun.apprunid.substring(0, 8)})</div>
            </div>
        </div>
    );
};

const NoAppRunsFound: React.FC = () => {
    return (
        <div className="flex flex-col items-center h-full p-6 text-center">
            <div className="grow" />
            <div className="flex flex-col items-center">
                <h3 className="text-primary text-lg font-medium mb-4">Getting Started with Outrig</h3>
                <p className="text-secondary mb-6">Add Outrig to your Go application with just a few lines of code:</p>
                <pre className="whitespace-pre bg-panel border border-border rounded-lg p-4 text-left text-sm text-primary overflow-auto w-full max-w-lg">
                    <code>
                        {`// Step 1: Import the package
import "github.com/outrigdev/outrig"

func main() {
    // Step 2: Initialize Outrig (set your application name)
    outrig.Init("app-name", nil)
    
    // Step 3: Optionally signal graceful shutdown
    defer outrig.AppDone()
    
    // Your application code here...
}`}
                    </code>
                </pre>
                <p className="text-secondary mt-6">Once you run your application, it will appear here automatically.</p>
            </div>
            <div className="grow-2" />
        </div>
    );
};

export const AppRunList: React.FC = () => {
    const unsortedAppRuns = useAtomValue(AppModel.appRunModel.appRuns);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // Sort app runs: running apps at the top, then by start time (newest first)
    const appRuns = useMemo(() => {
        return [...unsortedAppRuns].sort((a, b) => {
            // First sort by status (running at the top)
            if (a.status === "running" && b.status !== "running") return -1;
            if (a.status !== "running" && b.status === "running") return 1;

            // Then sort by start time (newest first)
            return b.starttime - a.starttime;
        });
    }, [unsortedAppRuns]);

    const handleAppRunClick = (appRunId: string) => {
        AppModel.selectAppRun(appRunId);
    };

    return (
        <div className="w-full h-full flex flex-col">
            <div className="flex-1 overflow-auto">
                {appRuns.length === 0 ? (
                    <NoAppRunsFound />
                ) : (
                    <div className="divide-y divide-border">
                        {appRuns.map((appRun) => (
                            <AppRunItem
                                key={appRun.apprunid}
                                appRun={appRun}
                                onClick={handleAppRunClick}
                                isSelected={appRun.apprunid === selectedAppRunId}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
