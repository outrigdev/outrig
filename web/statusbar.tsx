import { useAtomValue, useSetAtom } from "jotai";
import { Box, CircleDot, List, Wifi, WifiOff, PauseCircle } from "lucide-react";
import { AppModel } from "./appmodel";
import { Tooltip } from "./elements/tooltip";
import { useMemo } from "react";

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
    const appRuns = useAtomValue(AppModel.appRunModel.appRuns);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const setSelectedTab = useSetAtom(AppModel.selectedTab);

    // Find the selected app run
    const selectedAppRun = useMemo(() => {
        return appRuns.find(run => run.apprunid === selectedAppRunId);
    }, [appRuns, selectedAppRunId]);

    // Count running app runs
    const runningAppRunsCount = useMemo(() => {
        return appRuns.filter(run => run.status === "running").length;
    }, [appRuns]);

    // Determine which goroutine count to display based on app status
    const goroutineCount = useMemo(() => {
        if (!selectedAppRun) return 0;
        
        // For running apps, show active goroutines; otherwise show total goroutines
        return selectedAppRun.status === "running" 
            ? selectedAppRun.numactivegoroutines 
            : selectedAppRun.numtotalgoroutines;
    }, [selectedAppRun]);

    // Determine the tooltip text for goroutines
    const goroutineTooltip = useMemo(() => {
        if (!selectedAppRun) return "";
        
        if (selectedAppRun.status === "running") {
            return `${selectedAppRun.numactivegoroutines} Active GoRoutines (${selectedAppRun.numtotalgoroutines} Total)`;
        } else {
            return `${selectedAppRun.numtotalgoroutines} GoRoutines`;
        }
    }, [selectedAppRun]);

    return (
        <div className="h-6 bg-panel border-t border-border flex items-center justify-between px-2 text-xs text-secondary">
            <div className="flex items-center space-x-4">
                {selectedAppRun ? (
                    <>
                        <div className="flex items-center space-x-1">
                            <Box size={12} />
                            <span>{selectedAppRun.appname}</span>
                            <span className="text-muted">({selectedAppRun.apprunid.substring(0, 8)})</span>
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
                        <div className="flex items-center space-x-1 cursor-pointer" onClick={() => setSelectedTab("logs")}>
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
                </div>
            )}
        </div>
    );
}
