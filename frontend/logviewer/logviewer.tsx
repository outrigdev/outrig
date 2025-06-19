// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { LogVList } from "@/logvlist/logvlist";
import { SettingsModel } from "@/settings/settings-model";
import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { useAtomValue } from "jotai";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { LogViewerFilter } from "./logfilter";
import { LogLineComponent } from "./logline";
import { EmptyMessageDisplay, MarkedLinesIndicator, RefreshingModal, StreamingStatusBar } from "./logviewer-comps";
import { useLogViewerContextMenu } from "./logviewer-contextmenu";
import { LogViewerModel } from "./logviewer-model";

// LogList component for rendering the list of logs using LogVList
interface LogListProps {
    model: LogViewerModel;
}

const LogList = React.memo<LogListProps>(({ model }) => {
    const listContainerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const followOutput = useAtomValue(model.followOutput);
    const { contextMenu, handleContextMenu } = useLogViewerContextMenu(model);

    // Subscribe to atoms once at the LogList level
    const lineNumWidth = useAtomValue(model.lineNumberWidth);
    const logSettings = useAtomValue(SettingsModel.logsSettings);
    const appRunStartTime = useAtomValue(AppModel.appRunStartTimeAtom);

    // Memoize the lineSettings object so it only changes when the actual values change
    const lineSettings = useMemo(
        () => ({
            lineNumWidth,
            logSettings,
            appRunStartTime,
        }),
        [lineNumWidth, logSettings, appRunStartTime]
    );

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
        ({
            line,
            pageNum,
            lineIndex,
            onContextMenu,
        }: {
            line: LogLine;
            pageNum: number;
            lineIndex: number;
            onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void;
        }) => {
            return (
                <LogLineComponent
                    line={line}
                    model={model}
                    lineSettings={lineSettings}
                    pageNum={pageNum}
                    lineIndex={lineIndex}
                    onContextMenu={onContextMenu}
                />
            );
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

    return (
        <div ref={listContainerRef} className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight">
            {/* Always render LogVList, even during refresh */}
            <LogVList
                listAtom={model.listAtom}
                defaultItemHeight={15}
                lineComponent={lineComponent}
                containerHeight={dimensions.height} // Fallback height if dimensions not set yet
                onPageRequired={onPageRequired}
                onContextMenu={handleContextMenu}
                pinToBottomAtom={model.followOutput}
                vlistRef={model.vlistRef}
            />
            {contextMenu}
        </div>
    );
});
LogList.displayName = "LogList";

// Log content component
interface LogViewerContentProps {
    model: LogViewerModel;
}

const LogViewerContent = React.memo<LogViewerContentProps>(({ model }) => {
    const isRefreshing = useAtomValue(model.isRefreshing);
    const filteredLinesCount = useAtomValue(model.filteredItemCount);
    const totalLinesCount = useAtomValue(model.totalItemCount);
    const searchTerm = useAtomValue(model.searchTerm);
    const [showEmptyMessage, setShowEmptyMessage] = useState(false);

    // Set a timeout to show empty message after component mounts or when counts change
    useEffect(() => {
        if ((filteredLinesCount === 0 || totalLinesCount === 0) && !isRefreshing) {
            const timer = setTimeout(() => {
                setShowEmptyMessage(true);
            }, EmptyMessageDelayMs);

            return () => clearTimeout(timer);
        } else {
            setShowEmptyMessage(false);
        }
    }, [filteredLinesCount, totalLinesCount, isRefreshing]);

    // Get app running status
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(selectedAppRunId);
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const isAppRunning = appRunInfo?.isrunning || false;

    return (
        <div className="w-full h-full overflow-hidden flex-1 pt-2 relative flex flex-col">
            <MarkedLinesIndicator model={model} />

            {/* Main content area with logs */}
            <div className="flex-1 overflow-hidden mb-1">
                <LogList model={model} />
            </div>

            {isAppRunning && <StreamingStatusBar model={model} />}

            {isRefreshing && <RefreshingModal />}

            {!isRefreshing && showEmptyMessage && (
                <EmptyMessageDisplay
                    totalLinesCount={totalLinesCount}
                    filteredLinesCount={filteredLinesCount}
                    searchTerm={searchTerm}
                />
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
    if (!model) {
        return null;
    }

    return <LogViewerInternal key={props.appRunId} model={model} />;
});
LogViewer.displayName = "LogViewer";
