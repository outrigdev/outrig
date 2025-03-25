import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import { ArrowDown, ArrowDownCircle, Filter } from "lucide-react";
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
    searchRef: React.RefObject<HTMLInputElement>;
    className?: string;
}

export const LogViewerFilter = React.memo<LogViewerFilterProps>(({ model, searchRef, className }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredCount = useAtomValue(model.filteredItemCount);
    const searchedCount = useAtomValue(model.searchedItemCount);
    const totalCount = useAtomValue(model.totalItemCount);

    const handleKeyDown = useCallback(
        (e: React.KeyboardEvent) => {
            return keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
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

                if (checkKeyPressed(keyEvent, "Escape")) {
                    setSearch("");
                    return true;
                }

                return false;
            })(e);
        },
        [model, setSearch]
    );

    return (
        <div className={`py-1 px-1 border-b border-border ${className || ""}`}>
            <div className="flex items-center justify-between">
                <div className="flex items-center flex-grow">
                    {/* Line number space - 6 characters wide with right-aligned filter icon */}
                    <div className="select-none pr-2 text-muted w-12 text-right font-mono flex justify-end items-center">
                        <Filter
                            size={16}
                            className="text-muted"
                            fill="currentColor"
                            stroke="currentColor"
                            strokeWidth={1}
                        />
                    </div>

                    {/* Filter input */}
                    <input
                        ref={searchRef}
                        type="text"
                        placeholder="Filter logs..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
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
