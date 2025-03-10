import { useAtom, useAtomValue } from "jotai";
import { Filter } from "lucide-react";
import React, { JSX, useEffect, useRef, useState } from "react";
import { VariableSizeList as List } from "react-window";
import { AppModel } from "../appmodel";
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
        return null;
    }

    return (
        <div style={style}>
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

// Header component
interface LogViewerHeaderProps {
    model: LogViewerModel;
    className?: string;
}

const LogViewerHeader = React.memo<LogViewerHeaderProps>(({ model, className }) => {
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const appRuns = useAtomValue(AppModel.appRuns);

    // Find the selected app run
    const selectedAppRun = appRuns.find((run) => run.apprunid === selectedAppRunId);

    return (
        <div className={`py-2 px-4 border-b border-border ${className || ""}`}>
            <div className="flex justify-between items-center">
                <h2 className="text-lg font-semibold text-primary">
                    {selectedAppRun ? `Logs: ${selectedAppRun.appname}` : "Logs"}
                </h2>
                {selectedAppRun && <div className="text-xs text-muted">ID: {selectedAppRun.apprunid}</div>}
            </div>
        </div>
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
            <div className="flex items-center">
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
        </div>
    );
});

// Log content component
interface LogViewerContentProps {
    model: LogViewerModel;
}

const LogViewerContent = React.memo<LogViewerContentProps>(({ model }) => {
    const filteredLogLines = useAtomValue(model.filteredLogLines);
    const containerRef = useRef<HTMLDivElement>(null);
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
    }, []);

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

    return (
        <div ref={containerRef} className="w-full h-full overflow-hidden flex-1">
            {dimensions.height > 0 && filteredLogLines.length > 0 && (
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
            )}
        </div>
    );
});

// Main LogViewer component
export const LogViewer = React.memo<object>(() => {
    const model = useRef(new LogViewerModel()).current;
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const searchRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        // Load logs when the component mounts if an app run is selected
        if (selectedAppRunId) {
            AppModel.loadAppRunLogs(selectedAppRunId);
        }
    }, [selectedAppRunId]);

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
            <LogViewerHeader model={model} className="flex-shrink-0" />
            <LogViewerFilter model={model} searchRef={searchRef} className="flex-shrink-0" />

            {/* Subtle divider */}
            <div className="h-px bg-border flex-shrink-0"></div>

            <LogViewerContent model={model} />
        </div>
    );
});
