import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { LogVList } from "@/logvlist/logvlist";
import { useOutrigModel } from "@/util/hooks";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { cn } from "@/util/util";
import { getDefaultStore, useAtom, useAtomValue } from "jotai";
import { ArrowDown, ArrowDownCircle, Filter, X } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
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
    const padded = srcStr.padStart(6, " ");
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

// LogLineComponent for rendering individual log lines in LogVList
interface LogLineComponentProps {
    line: LogLine;
    model?: LogViewerModel;
}

const LogLineComponent = React.memo<LogLineComponentProps>(({ line, model }) => {
    // Subscribe to the version atom to trigger re-renders when marked lines change
    useAtomValue(model.markedLinesVersion);

    const handleLineNumberClick = useCallback(() => {
        model.toggleLineMarked(line.linenum);
    }, [model, line.linenum]);

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
LogLineComponent.displayName = "LogLineComponent";

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

const LogViewerFilter = React.memo<LogViewerFilterProps>(({ model, searchRef, className }) => {
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

// LogList component for rendering the list of logs using LogVList
interface LogListProps {
    model: LogViewerModel;
}

const LogList = React.memo<LogListProps>(({ model }) => {
    const listContainerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const followOutput = useAtomValue(model.followOutput);
    const isRefreshing = useAtomValue(model.isRefreshing);

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
            return <LogLineComponent line={line} model={model} />;
        },
        [model]
    );

    // Handle page required callback
    const onPageRequired = useCallback(
        (pageNum: number) => {
            model.onPageRequired(pageNum);
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
    const searchRef = useRef<HTMLInputElement>(null);
    const vlistRef = useRef<HTMLDivElement>(null);
    const searchTerm = useAtomValue(model.searchTerm);
    
    // Set the vlistRef in the model
    useEffect(() => {
        model.setVListRef(vlistRef);
    }, [model, vlistRef]);

    useEffect(() => {
        model.onSearchTermUpdate(searchTerm);
    }, [model, searchTerm]);

    // Focus the search input when the component mounts
    useEffect(() => {
        // Use a small timeout to ensure the input is ready
        const timer = setTimeout(() => {
            searchRef.current?.focus();
        }, 50);
        return () => clearTimeout(timer);
    }, []);

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
            <LogViewerFilter
                model={model}
                searchRef={searchRef}
                className="flex-shrink-0"
            />
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
