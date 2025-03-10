import { useAtomValue } from "jotai";
import { useEffect, useMemo } from "react";
import { AppModel } from "../appmodel";
import { Tag } from "../elements/tag";

export const AppRunList: React.FC = () => {
    const unsortedAppRuns = useAtomValue(AppModel.appRuns);

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

    useEffect(() => {
        // Load app runs when the component mounts
        AppModel.loadAppRuns();

        // Set up a refresh interval
        const intervalId = setInterval(() => {
            AppModel.loadAppRuns();
        }, 5000); // Refresh every 5 seconds

        return () => clearInterval(intervalId);
    }, []);

    const formatTimestamp = (timestamp: number): string => {
        const date = new Date(timestamp);
        return date.toLocaleString();
    };

    const handleAppRunClick = (appRunId: string) => {
        AppModel.selectAppRun(appRunId);
    };

    return (
        <div className="w-full h-full flex flex-col">
            <div className="py-2 px-4 border-b border-border">
                <h2 className="text-lg font-semibold text-primary">App Runs</h2>
            </div>

            <div className="flex-1 overflow-auto">
                {appRuns.length === 0 ? (
                    <div className="flex items-center justify-center h-full text-secondary">No app runs found</div>
                ) : (
                    <div className="divide-y divide-border">
                        {appRuns.map((appRun) => (
                            <div
                                key={appRun.apprunid}
                                className="p-4 hover:bg-buttonhover cursor-pointer"
                                onClick={() => handleAppRunClick(appRun.apprunid)}
                            >
                                <div className="flex justify-between items-center">
                                    <div className="font-medium text-primary">{appRun.appname}</div>
                                    <div className="text-xs text-secondary">
                                        {appRun.status === "running" ? (
                                            <Tag 
                                                label="Running" 
                                                variant="success" 
                                                isSelected={true} 
                                            />
                                        ) : appRun.status === "done" ? (
                                            <Tag 
                                                label="Done" 
                                                variant="info" 
                                                isSelected={true} 
                                            />
                                        ) : (
                                            <Tag 
                                                label="Disconnected" 
                                                variant="secondary" 
                                                isSelected={true} 
                                            />
                                        )}
                                    </div>
                                </div>
                                <div className="mt-1 text-sm text-secondary">
                                    Started: {formatTimestamp(appRun.starttime)}
                                </div>
                                <div className="mt-1 text-xs text-muted">ID: {appRun.apprunid}</div>
                                <div className="mt-1 text-xs text-muted">Logs: {appRun.numlogs}</div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
