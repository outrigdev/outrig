import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { SearchFilter } from "@/searchfilter/searchfilter";
import { checkKeyPressed } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import { ArrowDown, ArrowDownCircle } from "lucide-react";
import React, { useCallback } from "react";
import { LogViewerModel } from "./logviewer-model";

// Follow Button component
interface FollowButtonProps {
    model: LogViewerModel;
}

const FollowButton = React.memo<FollowButtonProps>(({ model }) => {
    const [followOutput, setFollowOutput] = useAtom(model.followOutput);

    const toggleFollow = useCallback(() => {
        const newFollowState = !followOutput;
        setFollowOutput(newFollowState);

        if (newFollowState) {
            model.scrollToBottom();
        }
    }, [followOutput, model, setFollowOutput]);

    return (
        <Tooltip content={followOutput ? "Tailing Log (Click to Disable)" : "Not Tailing Log (Click to Enable)"}>
            <button
                onClick={toggleFollow}
                className={`p-1 mr-1 rounded ${
                    followOutput
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                } cursor-pointer transition-colors`}
                aria-pressed={followOutput}
            >
                {followOutput ? <ArrowDownCircle size={16} /> : <ArrowDown size={16} />}
            </button>
        </Tooltip>
    );
});
FollowButton.displayName = "FollowButton";

// Filter component
interface LogViewerFilterProps {
    model: LogViewerModel;
    className?: string;
}

export const LogViewerFilter = React.memo<LogViewerFilterProps>(({ model, className }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredCount = useAtomValue(model.filteredItemCount);
    const searchedCount = useAtomValue(model.searchedItemCount);
    const totalCount = useAtomValue(model.totalItemCount);

    return (
        <div className={`py-1 px-1 border-b border-border ${className || ""}`}>
            <div className="flex items-center justify-between">
                {/* Use the SearchFilter component with a custom width for the icon */}
                <div className="flex items-center flex-grow">
                    <div className="w-2"></div> {/* Extra space for logs */}
                    <SearchFilter
                        value={search}
                        onValueChange={setSearch}
                        placeholder="Filter logs..."
                        autoFocus={true}
                        onOutrigKeyDown={(keyEvent) => {
                            if (checkKeyPressed(keyEvent, "Cmd:ArrowDown")) {
                                model.scrollToBottom();
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "Cmd:ArrowUp")) {
                                model.scrollToTop();
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "PageUp")) {
                                model.pageUp();
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "PageDown")) {
                                model.pageDown();
                                return true;
                            }
                            return false;
                        }}
                    />
                </div>

                {/* Search stats */}
                <Tooltip content={`${filteredCount} matched / ${searchedCount} searched / ${totalCount} ingested`}>
                    <div className="text-xs text-muted mr-2 select-none cursor-pointer">
                        {filteredCount}/{searchedCount}
                        {totalCount > searchedCount ? "+" : ""}
                    </div>
                </Tooltip>

                <FollowButton model={model} />
                <RefreshButton
                    isRefreshingAtom={model.isRefreshing}
                    onRefresh={() => model.refresh()}
                    tooltipContent="Refresh logs"
                />
            </div>
        </div>
    );
});
LogViewerFilter.displayName = "LogViewerFilter";
