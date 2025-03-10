import { useAtom, useAtomValue } from "jotai";
import { Filter, RefreshCw } from "lucide-react";
import React, { JSX, useEffect, useRef, useState } from "react";
import { VariableSizeList as List } from "react-window";
import { LogViewerModel } from "./logviewer-model";

// Utility functions
function formatLineNumber(num: number, width = 4) {
    return String(num).padStart(width, " ");
}

function formatTimestamp(ts: number, format: string = "HH:mm:ss.SSS") {
    const date = new Date(ts);
    const hh = date.getHours().toString().padStart(2, "0");
    const mm = date.getMinutes().toString().padStart(2, "0");
    const ss = date.getSeconds().toString().padStart(2, "0");
    const sss = date.getMilliseconds().toString().padStart(3, "0");
    return `${hh}:${mm}:${ss}.${sss}`;
}

function formatSource(source: string): JSX.Element {
    let srcStr = source || "";
    if (srcStr.startsWith("/dev/")) {
        srcStr = srcStr.slice(5);
    }
    const padded = srcStr.padEnd(6, " ");
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

// Individual log line component that also serves as a row for react-window
interface LogLineViewProps {
    line: LogLine;
    style?: React.CSSProperties;
}

const LogLineView = React.memo<LogLineViewProps>(({ line, style }) => {
    if (line == null) {
        return <div key="main" style={style}></div>;
    }
    return (
        <div key="main" style={style}>
            <div className="flex whitespace-nowrap hover:bg-buttonhover">
                <div className="select-none pr-2 text-muted w-12 text-right">{formatLineNumber(line.linenum, 4)}</div>
                <div>
                    <span className="text-secondary">{formatTimestamp(line.ts, "HH:mm:ss.SSS")}</span>{" "}
                    {formatSource(line.source)} <span className="text-primary">{line.msg}</span>
                </div>
            </div>
        </div>
    );
});

// Refresh Button component
interface RefreshButtonProps {
    model: LogViewerModel;
}

const RefreshButton = React.memo<RefreshButtonProps>(({ model }) => {
    const isRefreshing = useAtomValue(model.isRefreshing);
    const [isAnimating, setIsAnimating] = useState(false);

    const handleRefresh = () => {
        if (isRefreshing || isAnimating) return;

        // Start animation
        setIsAnimating(true);

        // Start refresh
        model.refresh();

        // End animation after 500ms
        setTimeout(() => {
            setIsAnimating(false);
        }, 500);
    };

    return (
        <button
            onClick={handleRefresh}
            className={`p-1 mr-1 rounded hover:bg-buttonhover text-muted hover:text-primary cursor-pointer ${isAnimating ? "refresh-spin" : ""}`}
            title="Refresh logs"
            disabled={isRefreshing || isAnimating}
        >
            <RefreshCw size={16} />
        </button>
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
                        placeholder="filter..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
                    />
                </div>
                <RefreshButton model={model} />
            </div>
        </div>
    );
});

// LogList component for rendering the virtualized list of logs
interface LogListProps {
    model: LogViewerModel;
    containerRef: React.RefObject<HTMLDivElement>;
}

const LogList = React.memo<LogListProps>(({ model, containerRef }) => {
    const filteredLogLines = useAtomValue(model.filteredLogLines);
    const listRef = useRef<List>(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

    // Default line height (for single-line logs)
    const DEFAULT_LINE_HEIGHT = 20;

    // Function to calculate item height - currently all items have the same height
    // but in the future this could vary based on content (stack traces, wrapping, etc.)
    const getItemHeight = (index: number) => {
        // For now, all items have the same height
        // In the future, this could analyze the log line to determine if it's a stack trace
        // or calculate height based on content length if wrapping is enabled
        return DEFAULT_LINE_HEIGHT;
    };

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

    // Reset list item size cache when filtered logs change
    useEffect(() => {
        if (listRef.current) {
            listRef.current.resetAfterIndex(0);
        }
    }, [filteredLogLines]);

    // Row renderer function that directly uses LogLineView
    const rowRenderer = ({ index, style }: { index: number; style: React.CSSProperties }) => {
        const line = filteredLogLines[index];
        if (!line) return null;
        return <LogLineView line={line} style={style} />;
    };

    if (dimensions.height === 0) {
        return null;
    }

    return (
        <div className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight px-1 pt-2">
            <List
                ref={listRef}
                height={dimensions.height}
                width="100%"
                itemCount={filteredLogLines.length}
                itemSize={getItemHeight}
                overscanCount={20}
                itemKey={(index) => filteredLogLines[index]?.linenum.toString() || index.toString()}
            >
                {rowRenderer}
            </List>
        </div>
    );
});

// Log content component
interface LogViewerContentProps {
    model: LogViewerModel;
}

const LogViewerContent = React.memo<LogViewerContentProps>(({ model }) => {
    const isRefreshing = useAtomValue(model.isRefreshing);
    const filteredLogLines = useAtomValue(model.filteredLogLines);
    const containerRef = useRef<HTMLDivElement>(null);

    return (
        <div ref={containerRef} className="w-full h-full overflow-hidden flex-1">
            {isRefreshing && (
                <div className="w-full h-full flex items-center justify-center">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing logs...</span>
                    </div>
                </div>
            )}

            {!isRefreshing && filteredLogLines.length === 0 && (
                <div className="w-full h-full flex items-center justify-center text-muted">No logs found</div>
            )}

            {!isRefreshing && filteredLogLines.length > 0 && <LogList model={model} containerRef={containerRef} />}
        </div>
    );
});

interface LogViewerProps {
    appRunId: string;
}

export const LogViewer = React.memo<LogViewerProps>((props: LogViewerProps) => {
    const { appRunId } = props;
    const model = useRef(new LogViewerModel(appRunId)).current;
    const searchRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        model.loadAppRunLogs();
    }, [model]);

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

            {/* Subtle divider */}
            <div className="h-px bg-border flex-shrink-0"></div>

            <LogViewerContent model={model} />
        </div>
    );
});
