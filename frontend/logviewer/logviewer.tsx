// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { CopyButton } from "@/elements/copybutton";
import { LogVList } from "@/logvlist/logvlist";
import { LogSettings, SettingsModel } from "@/settings/settings-model";
import { useOutrigModel } from "@/util/hooks";
import { useAtomValue } from "jotai";
import { X } from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { LogViewerFilter } from "./logfilter";
import { LogLineComponent } from "./logline";
import { LogViewerModel } from "./logviewer-model";

// Interface for combined log line settings
interface LogLineSettings {
    lineNumWidth: number;
    logSettings: LogSettings;
    appRunStartTime: number | null;
}

// LogList component for rendering the list of logs using LogVList
interface LogListProps {
    model: LogViewerModel;
}

const LogList = React.memo<LogListProps>(({ model }) => {
    const listContainerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const followOutput = useAtomValue(model.followOutput);
    const isRefreshing = useAtomValue(model.isRefreshing);
    
    // Subscribe to atoms once at the LogList level
    const lineNumWidth = useAtomValue(model.lineNumberWidth);
    const logSettings = useAtomValue(SettingsModel.logsSettings);
    const appRunStartTime = useAtomValue(AppModel.appRunStartTimeAtom);
    
    // Memoize the lineSettings object so it only changes when the actual values change
    const lineSettings = useMemo(() => ({
        lineNumWidth,
        logSettings,
        appRunStartTime
    }), [lineNumWidth, logSettings, appRunStartTime]);

    // Prevent default smooth scrolling for PageUp/PageDown when focus is in the list
    useEffect(() => {
        if (!model.vlistRef.current) return;

        // Capture the current value of the ref
        const currentContainer = model.vlistRef.current;

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "PageUp") {
                e.preventDefault();
                model.pageUp();
            } else if (e.key === "PageDown") {
                e.preventDefault();
                model.pageDown();
            }
        };

        currentContainer.addEventListener("keydown", handleKeyDown);
        return () => {
            currentContainer.removeEventListener("keydown", handleKeyDown);
        };
    }, [model]);

    // We don't need to handle followOutput changes here as it's handled by LogVList

    // Handle visibility changes (when switching tabs)
    useEffect(() => {
        const handleVisibilityChange = () => {
            if (!document.hidden && followOutput) {
                // When tab becomes visible and follow mode is enabled, scroll to bottom
                model.scrollToBottom();
            }
        };

        document.addEventListener("visibilitychange", handleVisibilityChange);
        return () => {
            document.removeEventListener("visibilitychange", handleVisibilityChange);
        };
    }, [followOutput, model]);

    // Update dimensions when the list container is resized
    useEffect(() => {
        if (!listContainerRef.current) return;

        const updateDimensions = () => {
            if (listContainerRef.current) {
                setDimensions({
                    width: listContainerRef.current.offsetWidth,
                    height: listContainerRef.current.offsetHeight,
                });
            }
        };

        // Initial dimensions
        updateDimensions();

        // Set up resize observer
        const observedElement = listContainerRef.current;
        const resizeObserver = new ResizeObserver(updateDimensions);
        resizeObserver.observe(observedElement);

        return () => {
            resizeObserver.unobserve(observedElement);
            resizeObserver.disconnect();
        };
    }, []);

    // We don't need to handle scroll position changes here as it's handled in LogVList

    // Create the line component for LogVList
    const lineComponent = useCallback(
        ({ line }: { line: LogLine }) => {
            return <LogLineComponent line={line} model={model} lineSettings={lineSettings} />;
        },
        [model, lineSettings]
    );

    // Handle page required callback
    const onPageRequired = useCallback(
        (pageNum: number, load: boolean) => {
            model.onPageRequired(pageNum, load);
        },
        [model]
    );

    console.log("LogList render", dimensions, "isRefreshing:", isRefreshing);

    return (
        <div ref={listContainerRef} className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight">
            {/* Always render LogVList, even during refresh */}
            <LogVList
                listAtom={model.listAtom}
                defaultItemHeight={15}
                lineComponent={lineComponent}
                containerHeight={dimensions.height} // Fallback height if dimensions not set yet
                onPageRequired={onPageRequired}
                pinToBottomAtom={model.followOutput}
                vlistRef={model.vlistRef}
            />
        </div>
    );
});
LogList.displayName = "LogList";

// Marked Lines Indicator component
interface MarkedLinesIndicatorProps {
    model: LogViewerModel;
}

const MarkedLinesIndicator = React.memo<MarkedLinesIndicatorProps>(({ model }) => {
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
        <div className="absolute top-0 right-0 flex items-center bg-accent text-white dark:text-black rounded-bl-md px-2 py-1 text-xs z-10">
            <span className="font-medium">
                {markedCount} {markedCount === 1 ? "line" : "lines"} marked
            </span>
            <CopyButton
                className="ml-2 text-white dark:text-black"
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

// Log content component
interface LogViewerContentProps {
    model: LogViewerModel;
}

const LogViewerContent = React.memo<LogViewerContentProps>(({ model }) => {
    const isRefreshing = useAtomValue(model.isRefreshing);
    const isLoading = useAtomValue(model.isLoading);
    const filteredLinesCount = useAtomValue(model.filteredItemCount);

    return (
        <div className="w-full h-full overflow-hidden flex-1 pt-2 px-1 relative">
            <MarkedLinesIndicator model={model} />

            {/* Always render LogList */}
            <LogList model={model} />

            {/* Small centered refreshing modal with improved styling */}
            {isRefreshing && (
                <>
                    {/* Semi-transparent backdrop with minimal blur */}
                    <div className="absolute inset-0 bg-background/20 backdrop-blur-[1px] z-10"></div>

                    {/* Refreshing modal */}
                    <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-[300px] h-[120px] bg-panel border border-border rounded-md shadow-lg flex items-center justify-center z-20">
                        <div className="text-primary font-medium">Data Refreshed</div>
                    </div>
                </>
            )}

            {!isRefreshing && filteredLinesCount === 0 && (
                <div className="absolute inset-0 w-full h-full flex items-center justify-center bg-background/80">
                    <span className="text-muted">no matching lines</span>
                </div>
            )}
        </div>
    );
});
LogViewerContent.displayName = "LogViewerContent";

interface LogViewerInternalProps {
    model: LogViewerModel;
}

const LogViewerInternal = React.memo<LogViewerInternalProps>(({ model }) => {
    const vlistRef = useRef<HTMLDivElement>(null);
    const searchTerm = useAtomValue(model.searchTerm);

    // Set the vlistRef in the model
    useEffect(() => {
        model.setVListRef(vlistRef);
    }, [model, vlistRef]);

    useEffect(() => {
        model.onSearchTermUpdate(searchTerm);
    }, [model, searchTerm]);

    return (
        <div className="w-full h-full flex flex-col overflow-hidden">
            <LogViewerFilter model={model} className="flex-shrink-0" />
            <div className="h-px bg-border flex-shrink-0"></div>
            <LogViewerContent model={model} />
        </div>
    );
});
LogViewerInternal.displayName = "LogViewerInternal";

interface LogViewerProps {
    appRunId: string;
}

export const LogViewer = React.memo<LogViewerProps>((props: LogViewerProps) => {
    const model = useOutrigModel(LogViewerModel, props.appRunId);

    console.log("Render logviewer", props.appRunId, model);

    if (!model) {
        return null;
    }

    return <LogViewerInternal key={props.appRunId} model={model} />;
});
LogViewer.displayName = "LogViewer";
