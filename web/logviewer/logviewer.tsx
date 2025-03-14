import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { cn } from "@/util/util";
import { getDefaultStore, useAtom, useAtomValue } from "jotai";
import { ArrowDown, ArrowDownCircle, Filter, X } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { ListRange, Virtuoso, VirtuosoHandle } from "react-virtuoso";
import { LogViewerModel } from "./logviewer-model";

// Utility functions
function formatLineNumber(num: number, width = 4) {
    return String(num).padStart(width, " ");
}

function formatMarkedLineNumber(num: number, isMarked: boolean, width = 4): React.ReactNode {
    const paddedNum = String(num).padStart(width, " ");
    if (isMarked) {
        return (
            <span className="text-primary">
                <span className="text-accent">•</span> {paddedNum}
            </span>
        );
    }
    return <> {paddedNum}</>;
}

function formatTimestamp(ts: number, format: string = "HH:mm:ss.SSS") {
    const date = new Date(ts);
    const hh = date.getHours().toString().padStart(2, "0");
    const mm = date.getMinutes().toString().padStart(2, "0");
    const ss = date.getSeconds().toString().padStart(2, "0");
    const sss = date.getMilliseconds().toString().padStart(3, "0");
    return `${hh}:${mm}:${ss}.${sss}`;
}

function formatSource(source: string): React.ReactNode {
    let srcStr = source || "";
    if (srcStr.startsWith("/dev/")) {
        srcStr = srcStr.slice(5);
    }
    const padded = srcStr.padEnd(6, " ");
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

// LogLineItem component for rendering individual log lines
interface LogLineItemProps {
    index: number;
    model: LogViewerModel;
}

const LogLineItem = React.memo<LogLineItemProps>(({ index, model }) => {
    const logLineAtom = useRef(model.getLogIndexAtom(index)).current;
    const line = useAtomValue(logLineAtom);
    // Subscribe to the version atom to trigger re-renders when marked lines change
    useAtomValue(model.markedLinesVersion);

    const handleLineNumberClick = useCallback(() => {
        if (line == null) return;
        model.toggleLineMarked(line.linenum);
    }, [model, line]);

    if (line == null) {
        return (
            <div className="flex hover:bg-buttonhover select-none" style={{ height: 15 }}>
                <div className="pr-2 text-muted w-12 text-right flex-shrink-0"></div>
                <div className="flex-1 min-w-0">
                    <span className="text-secondary">...</span>
                </div>
            </div>
        );
    }

    const isMarked = model.isLineMarked(line.linenum);

    return (
        <div className={cn("flex text-muted select-none", isMarked ? "bg-accentbg/20" : "hover:bg-buttonhover")}>
            <div
                className={cn(
                    "w-12 text-right flex-shrink-0 cursor-pointer",
                    isMarked ? "text-accent" : "hover:text-primary"
                )}
                onClick={handleLineNumberClick}
            >
                {formatMarkedLineNumber(line.linenum, isMarked, 4)}
            </div>
            <div className="text-secondary flex-shrink-0 pl-2">{formatTimestamp(line.ts, "HH:mm:ss.SSS")}</div>
            <div className="pl-2">{formatSource(line.source)}</div>
            <div className="flex-1 min-w-0 pl-2 select-text">
                <span className="text-primary break-all overflow-hidden whitespace-pre">{line.msg}</span>
            </div>
        </div>
    );
});

// EOF component
const EofItem = React.memo(() => (
    <div className="flex py-3">
        <div className="select-none pr-2 text-muted w-12 text-right flex-shrink-0"></div>
        <div className="flex-1 min-w-0">
            <span className="text-muted">(end of log stream)</span>
        </div>
    </div>
));

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

// Filter component
interface LogViewerFilterProps {
    model: LogViewerModel;
    searchRef: React.RefObject<HTMLInputElement>;
    className?: string;
}

const LogViewerFilter = React.memo<LogViewerFilterProps>(({ model, searchRef, className }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredCount = useAtomValue(model.filteredItemCount);
    const totalCount = useAtomValue(model.totalItemCount);

    const handleKeyDown = useCallback(
        keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
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
        }),
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
                <div className="text-xs text-muted mr-2 select-none">
                    {filteredCount}/{totalCount}
                </div>

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

// LogList component for rendering the virtualized list of logs
interface LogListProps {
    model: LogViewerModel;
}

const LogList = React.memo<LogListProps>(({ model }) => {
    // Create a ref for the Virtuoso component
    const virtuosoRef = useRef<VirtuosoHandle>(null);

    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const filteredItemCount = useAtomValue(model.filteredItemCount);
    const followOutput = useAtomValue(model.followOutput);
    const containerRef = useRef<HTMLDivElement>(null);

    // Set the virtuoso ref in the model when it changes
    useEffect(() => {
        model.setVirtuosoRef(virtuosoRef);
    }, [model]);

    // Handle followOutput changes
    useEffect(() => {
        if (followOutput) {
            model.scrollToBottom();
        }
    }, [followOutput, model]);

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

    // Update dimensions when the container is resized
    useEffect(() => {
        if (!containerRef.current) return;

        const updateDimensions = () => {
            if (containerRef.current) {
                setDimensions({
                    width: containerRef.current.offsetWidth,
                    height: containerRef.current.offsetHeight,
                });
            }
        };

        // Initial dimensions
        updateDimensions();

        // Set up resize observer
        const observedElement = containerRef.current;
        const resizeObserver = new ResizeObserver(updateDimensions);
        resizeObserver.observe(observedElement);

        return () => {
            resizeObserver.unobserve(observedElement);
            resizeObserver.disconnect();
        };
    }, [containerRef]);

    const onRangeChanged = useCallback(
        (range: ListRange) => {
            model.setRenderedRange(range.startIndex, range.endIndex);
        },
        [model]
    );

    // Handle scroll position changes to update follow mode
    const handleAtBottomStateChange = useCallback(
        (isAtBottom: boolean) => {
            // Only update follow mode if it's different from current state
            const currentFollowMode = getDefaultStore().get(model.followOutput);
            if (currentFollowMode !== isAtBottom) {
                getDefaultStore().set(model.followOutput, isAtBottom);
            }
        },
        [model]
    );

    // Item renderer function for Virtuoso
    const itemRenderer = useCallback(
        (index: number) => {
            return <LogLineItem key={index} index={index} model={model} />;
        },
        [model]
    );

    let listElem = (
        <Virtuoso
            ref={virtuosoRef}
            style={{ height: dimensions.height, width: "100%" }}
            totalCount={filteredItemCount}
            itemContent={itemRenderer}
            followOutput={followOutput}
            initialTopMostItemIndex={followOutput ? filteredItemCount : undefined}
            overscan={200}
            defaultItemHeight={15}
            rangeChanged={onRangeChanged}
            atBottomStateChange={handleAtBottomStateChange}
        />
    );

    return (
        <div ref={containerRef} className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight">
            {dimensions.height > 0 ? listElem : null}
        </div>
    );
});

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
        <div className="absolute top-0 right-0 flex items-center bg-accent text-black rounded-bl-md px-2 py-1 text-xs z-10">
            <span className="font-medium">
                {markedCount} {markedCount === 1 ? "line" : "lines"} marked
            </span>
            <CopyButton
                className="ml-2"
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

            {isRefreshing && (
                <div className="w-full h-full flex items-center justify-center">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing logs...</span>
                    </div>
                </div>
            )}

            {!isRefreshing && filteredLinesCount === 0 && (
                <div className="w-full h-full flex items-center justify-center text-muted">no matching lines</div>
            )}

            {!isRefreshing && filteredLinesCount > 0 && <LogList model={model} />}
        </div>
    );
});

interface LogViewerInternalProps {
    model: LogViewerModel;
}

const LogViewerInternal = React.memo<LogViewerInternalProps>(({ model }) => {
    const searchRef = useRef<HTMLInputElement>(null);
    const searchTerm = useAtomValue(model.searchTerm);

    useEffect(() => {
        model.onSearchTermUpdate(searchTerm);
    }, [model, searchTerm]);

    useEffect(() => {
        // on window focus, focus the search input
        const onFocus = () => {
            searchRef.current?.focus();
        };
        window.addEventListener("focus", onFocus);
        return () => {
            window.removeEventListener("focus", onFocus);
        };
    }, []);

    return (
        <div className="w-full h-full flex flex-col overflow-hidden">
            <LogViewerFilter model={model} searchRef={searchRef} className="flex-shrink-0" />
            <div className="h-px bg-border flex-shrink-0"></div>
            <LogViewerContent model={model} />
        </div>
    );
});

interface LogViewerProps {
    appRunId: string;
}

export const LogViewer = React.memo<LogViewerProps>((props: LogViewerProps) => {
    const modelRef = useRef<LogViewerModel>(null);
    const [, setForceUpdate] = useState({});

    console.log("Render logviewer", props.appRunId, modelRef.current);

    useEffect(() => {
        if (!modelRef.current) {
            modelRef.current = new LogViewerModel(props.appRunId);
            setForceUpdate({});
        }
        return () => {
            if (!modelRef.current) {
                return;
            }
            modelRef.current.dispose();
            modelRef.current = null;
        };
    }, [props.appRunId]);

    if (!modelRef.current) {
        return null;
    }

    return <LogViewerInternal key={props.appRunId} model={modelRef.current} />;
});
