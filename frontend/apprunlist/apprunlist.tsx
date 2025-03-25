import { useAtomValue } from "jotai";
import { CircleDot, Clock, Eye, List } from "lucide-react";
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
        <a
            href={appRunUrl}
            className={cn(
                "p-4 hover:bg-buttonhover cursor-pointer block",
                isSelected && "bg-buttonhover border-l-4 border-l-accent"
            )}
            onClick={(e) => {
                // Prevent default navigation for left-click
                e.preventDefault();
                onClick(appRun.apprunid);
            }}
        >
            <div className="flex justify-between items-center">
                <div className="font-medium text-primary">{appRun.appname}</div>
                <div className="text-xs text-secondary">
                    <AppRunStatusTag status={appRun.status} />
                </div>
            </div>
            <div className="mt-1 text-sm text-secondary">
                Started {appRun.status === "running" ? "Running" : formatRelativeTime(appRun.starttime)}
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
                    <span>{appRun.status === "running" ? appRun.numactivegoroutines : appRun.numtotalgoroutines}</span>
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
        </a>
    );
};

const NoAppRunsFound: React.FC = () => {
    return <div className="flex items-center justify-center h-full text-secondary">no app runs found</div>;
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
