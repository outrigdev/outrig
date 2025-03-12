import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import { CaseSensitive, Code, Filter, RefreshCw, Search, Sparkles } from "lucide-react";
import React, { JSX, useCallback, useEffect, useRef, useState } from "react";
import { VariableSizeList as List, ListOnItemsRenderedProps } from "react-window";
import { LogViewerModel, SearchType } from "./logviewer-model";

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
    lineIndex: number;
    style: React.CSSProperties;
    model: LogViewerModel;
}

const LogLineView = React.memo<LogLineViewProps>(({ lineIndex, model, style }) => {
    const logLineAtom = useRef(model.getLogIndexAtom(lineIndex)).current;
    const line = useAtomValue(logLineAtom);
    if (line == null) {
        return (
            <div key="main" style={style}>
                <div className="flex whitespace-nowrap hover:bg-buttonhover">
                    <div className="select-none pr-2 text-muted w-12 text-right"></div>
                    <div>
                        <span className="text-secondary">...</span>
                    </div>
                </div>
            </div>
        );
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

const LogLineEofView = React.memo<LogLineViewProps>(({ lineIndex, model, style }) => {
    return (
        <div key="main" style={style}>
            <div className="flex whitespace-nowrap hover:bg-buttonhover">
                <div className="select-none pr-2 text-muted w-12 text-right"></div>
                <div className="text-secondary">-- EOF --</div>
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

// Search Type Selector component
interface SearchTypeSelectorProps {
    model: LogViewerModel;
}

const SearchTypeSelector = React.memo<SearchTypeSelectorProps>(({ model }) => {
    const [searchType, setSearchType] = useAtom(model.searchType);

    const searchTypes: { value: SearchType; label: string; icon: JSX.Element }[] = [
        {
            value: "exact",
            label: "Exact Match (case-insensitive)",
            icon: <Search size={14} className="mr-1" />,
        },
        {
            value: "exactcase",
            label: "Exact Match (case-sensitive)",
            icon: <CaseSensitive size={14} className="mr-1" />,
        },
        {
            value: "regexp",
            label: "Regular Expression",
            icon: <Code size={14} className="mr-1" />,
        },
        {
            value: "fzf",
            label: "Fuzzy Search",
            icon: <Sparkles size={14} className="mr-1" />,
        },
    ];

    const handleSearchTypeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        setSearchType(e.target.value as SearchType);
    };

    return (
        <div className="flex items-center mr-2">
            <select
                value={searchType}
                onChange={handleSearchTypeChange}
                className="bg-transparent text-primary text-xs border border-border rounded px-1 py-0.5"
            >
                {searchTypes.map((type) => (
                    <option key={type.value} value={type.value}>
                        {type.label}
                    </option>
                ))}
            </select>
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

            return false;
        }),
        [model]
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
                        placeholder="filter..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
                    />
                </div>

                {/* Search type selector */}
                <SearchTypeSelector model={model} />

                {/* Search stats */}
                <div className="text-xs text-muted mr-2 select-none">
                    {filteredCount}/{totalCount}
                </div>

                <RefreshButton model={model} />
            </div>
        </div>
    );
});

// LogList component for rendering the virtualized list of logs
interface LogListProps {
    model: LogViewerModel;
}

const LogList = React.memo<LogListProps>(({ model }) => {
    // Create a ref for the List component
    const listRef = useRef<List>(null);

    // Set the list ref in the model when it changes
    useEffect(() => {
        model.setListRef(listRef);
    }, [model]);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const filteredItemCount = useAtomValue(model.filteredItemCount);
    const containerRef = useRef<HTMLDivElement>(null);

    // Default line height (for single-line logs)
    const DEFAULT_LINE_HEIGHT = 15;

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

    const onRenderedCallback = useCallback(
        ({ overscanStartIndex, overscanStopIndex }: ListOnItemsRenderedProps) => {
            model.setRenderedRange(overscanStartIndex, overscanStopIndex);
        },
        [model]
    );

    // Row renderer function that directly uses LogLineView
    const rowRenderer = ({ index, style }: { index: number; style: React.CSSProperties }) => {
        if (index === filteredItemCount) {
            return <LogLineEofView key={index} lineIndex={index} model={model} style={style} />;
        }
        return <LogLineView key={index} lineIndex={index} model={model} style={style} />;
    };

    let listElem = (
        <List
            ref={listRef}
            height={dimensions.height}
            width="100%"
            itemCount={filteredItemCount + 1}
            itemSize={getItemHeight}
            overscanCount={20}
            onItemsRendered={onRenderedCallback}
        >
            {rowRenderer}
        </List>
    );

    return (
        <div ref={containerRef} className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight">
            {dimensions.height > 0 ? listElem : null}
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
        <div className="w-full h-full overflow-hidden flex-1 pt-2 px-1">
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
    const searchType = useAtomValue(model.searchType);

    useEffect(() => {
        model.onSearchTermUpdate(searchTerm, searchType);
    }, [model, searchTerm, searchType]);

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
    const [model, setModel] = useState<LogViewerModel>(null);
    useEffect(() => {
        const model = new LogViewerModel(props.appRunId);
        setModel(model);
        return () => {
            model.dispose();
        };
    }, [props.appRunId]);
    if (!model) {
        return null;
    }
    return <LogViewerInternal key={props.appRunId} model={model} />;
});
