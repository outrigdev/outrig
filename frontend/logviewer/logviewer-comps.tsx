// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { CopyButton } from "@/elements/copybutton";
import { Tooltip } from "@/elements/tooltip";
import { getDefaultStore, useAtom, useAtomValue } from "jotai";
import { ArrowDown, ArrowDownCircle, Wifi, WifiOff, X } from "lucide-react";
import React from "react";
import { LogViewerModel } from "./logviewer-model";

// Streaming status bar component
interface StreamingStatusBarProps {
    model: LogViewerModel;
}

export const StreamingStatusBar = React.memo<StreamingStatusBarProps>(({ model }) => {
    const isStreaming = useAtomValue(model.isStreaming);
    const [followOutput, setFollowOutput] = useAtom(model.followOutput);

    // Toggle streaming handler
    const handleToggleStreaming = () => {
        getDefaultStore().set(model.isStreaming, !isStreaming);
    };

    // Toggle follow output handler
    const handleToggleFollow = () => {
        setFollowOutput(!followOutput);
        if (!followOutput) {
            model.scrollToBottom();
        }
    };

    return (
        <div className="w-full flex items-center h-6 border-t border-border bg-primary/8">
            {/* Left side - Streaming button */}
            <div className="flex-grow basis-0 flex justify-end items-center">
                <Tooltip
                    content={
                        isStreaming
                            ? "Streaming New Log Lines (Click to Disable)"
                            : "Not Streaming New Log Lines (Click to Enable)"
                    }
                >
                    <button
                        onClick={handleToggleStreaming}
                        className={`flex items-center gap-2 px-2 py-0.5 rounded ${
                            isStreaming ? "text-primary" : "text-muted"
                        } hover:bg-primary/10 cursor-pointer transition-colors`}
                        aria-pressed={isStreaming}
                    >
                        {isStreaming ? <Wifi size={14} /> : <WifiOff size={14} />}
                        <span className="text-xs">{isStreaming ? "Streaming" : "Not Streaming"}</span>
                    </button>
                </Tooltip>
            </div>

            {/* Center divider */}
            <div className="mx-4 text-border">|</div>

            {/* Right side - Pin button */}
            <div className="flex-grow basis-0 flex justify-start items-center">
                <Tooltip
                    content={
                        followOutput ? "Pinned to Bottom (Click to Disable)" : "Not Pinned to Bottom (Click to Enable)"
                    }
                >
                    <button
                        onClick={handleToggleFollow}
                        className={`flex items-center gap-2 px-2 py-0.5 rounded ${
                            followOutput ? "text-primary" : "text-muted"
                        } hover:bg-primary/10 cursor-pointer transition-colors`}
                        aria-pressed={followOutput}
                    >
                        {followOutput ? <ArrowDownCircle size={14} /> : <ArrowDown size={14} />}
                        <span className="text-xs">{followOutput ? "Pinned" : "Not Pinned"}</span>
                    </button>
                </Tooltip>
            </div>
        </div>
    );
});
StreamingStatusBar.displayName = "StreamingStatusBar";

// Refreshing modal component
export const RefreshingModal = React.memo(() => {
    return (
        <>
            {/* Semi-transparent backdrop with minimal blur */}
            <div className="absolute inset-0 bg-background/20 backdrop-blur-[1px] z-10"></div>

            {/* Refreshing modal */}
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-[300px] h-[120px] bg-panel border border-border rounded-md shadow-lg flex items-center justify-center z-20">
                <div className="text-primary font-medium">Data Refreshed</div>
            </div>
        </>
    );
});
RefreshingModal.displayName = "RefreshingModal";

// Empty message display component
interface EmptyMessageDisplayProps {
    totalLinesCount: number;
    filteredLinesCount: number;
    searchTerm: string;
}

export const EmptyMessageDisplay = React.memo<EmptyMessageDisplayProps>(
    ({ totalLinesCount, filteredLinesCount, searchTerm }) => {
        return (
            <div className="absolute inset-0 w-full h-full flex items-center justify-center bg-background/80">
                {totalLinesCount === 0 ? (
                    <span className="text-muted">no log lines</span>
                ) : filteredLinesCount === 0 && searchTerm ? (
                    <span className="text-muted">no matching lines</span>
                ) : null}
            </div>
        );
    }
);
EmptyMessageDisplay.displayName = "EmptyMessageDisplay";

// Marked Lines Indicator component
interface MarkedLinesIndicatorProps {
    model: LogViewerModel;
}

export const MarkedLinesIndicator = React.memo<MarkedLinesIndicatorProps>(({ model }) => {
    // Subscribe to the version atom to trigger re-renders when marked lines change
    useAtomValue(model.markedLinesVersion);
    const markedCount = model.getMarkedLinesCount();

    if (markedCount === 0) {
        return null;
    }

    const handleClearMarks = () => {
        model.clearMarkedLines();
    };

    const handleCopyMarkedLines = async () => {
        await model.copyMarkedLinesToClipboard();
    };

    return (
        <div className="absolute bottom-0 right-0 flex items-center bg-accentbg text-primary dark:text-black rounded-tl-md px-2 py-1 text-xs z-10">
            <span className="font-medium">
                {markedCount} {markedCount === 1 ? "line" : "lines"} marked
            </span>
            <CopyButton
                className="ml-2 text-primary dark:text-black"
                size={14}
                tooltipText="Copy marked lines"
                successTooltipText="Copied!"
                variant="primary"
                onCopy={handleCopyMarkedLines}
            />
            <button
                onClick={handleClearMarks}
                className="ml-2 hover:text-black/70 cursor-pointer"
                aria-label="Clear marked lines"
            >
                <X size={14} />
            </button>
        </div>
    );
});
MarkedLinesIndicator.displayName = "MarkedLinesIndicator";
