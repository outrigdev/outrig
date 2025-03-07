import { useSetAtom } from "jotai";
import { Box, CircleDot, List, Wifi } from "lucide-react";
import { AppModel } from "./appmodel";
import { Tooltip } from "./elements/tooltip";

export function StatusBar() {
    const numGoRoutines = 24;
    const numLogLines = 1083;
    const setSelectedTab = useSetAtom(AppModel.selectedTab);

    return (
        <div className="h-6 bg-panel border-t border-border flex items-center justify-between px-2 text-xs text-secondary">
            <div className="flex items-center space-x-4">
                <div className="flex items-center space-x-1">
                    <Box size={12} />
                    <span>appname</span>
                </div>
                <div className="flex items-center space-x-1">
                    <Wifi size={12} />
                    <span>Connected</span>
                </div>
            </div>
            <div className="flex items-center space-x-4">
                <Tooltip content={`${numLogLines} Log Lines`} placement="bottom">
                    <div className="flex items-center space-x-1 cursor-pointer" onClick={() => setSelectedTab("logs")}>
                        <List size={12} />
                        <span>1083</span>
                    </div>
                </Tooltip>
                <Tooltip content={`${numGoRoutines} GoRoutines`} placement="bottom">
                    <div
                        className="flex items-center space-x-1 cursor-pointer"
                        onClick={() => setSelectedTab("goroutines")}
                    >
                        <CircleDot size={12} />
                        <span>24</span>
                    </div>
                </Tooltip>
            </div>
        </div>
    );
}
