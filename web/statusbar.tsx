import { useAtomValue, useSetAtom } from "jotai";
import { Box, CircleDot, List, Wifi, WifiOff, PauseCircle } from "lucide-react";
import { AppModel } from "./appmodel";
import { Tooltip } from "./elements/tooltip";

function ConnectionStatus() {
    const appStatus = useAtomValue(AppModel.appStatus);
    
    let icon;
    let displayName;
    
    switch (appStatus) {
        case "connected":
            icon = <Wifi size={12} />;
            displayName = "Connected";
            break;
        case "disconnected":
            icon = <WifiOff size={12} />;
            displayName = "Disconnected";
            break;
        case "paused":
            icon = <PauseCircle size={12} />;
            displayName = "Paused";
            break;
        default:
            icon = <Wifi size={12} />;
            displayName = "Connected";
    }
    
    return (
        <div className="flex items-center space-x-1">
            {icon}
            <span>{displayName}</span>
        </div>
    );
}

export function StatusBar() {
    const numGoRoutines = useAtomValue(AppModel.numGoRoutines);
    const numLogLines = useAtomValue(AppModel.numLogLines);
    const setSelectedTab = useSetAtom(AppModel.selectedTab);

    return (
        <div className="h-6 bg-panel border-t border-border flex items-center justify-between px-2 text-xs text-secondary">
            <div className="flex items-center space-x-4">
                <div className="flex items-center space-x-1">
                    <Box size={12} />
                    <span>appname</span>
                </div>
                <ConnectionStatus />
            </div>
            <div className="flex items-center space-x-4">
                <Tooltip content={`${numLogLines} Log Lines`} placement="bottom">
                    <div className="flex items-center space-x-1 cursor-pointer" onClick={() => setSelectedTab("logs")}>
                        <List size={12} />
                        <span>{numLogLines}</span>
                    </div>
                </Tooltip>
                <Tooltip content={`${numGoRoutines} GoRoutines`} placement="bottom">
                    <div
                        className="flex items-center space-x-1 cursor-pointer"
                        onClick={() => setSelectedTab("goroutines")}
                    >
                        <CircleDot size={12} />
                        <span>{numGoRoutines}</span>
                    </div>
                </Tooltip>
            </div>
        </div>
    );
}
