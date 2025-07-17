// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { cn } from "@/util/util";
import {
    createColumnHelper,
    flexRender,
    getCoreRowModel,
    getSortedRowModel,
    useReactTable,
} from "@tanstack/react-table";
import { getDefaultStore, useAtomValue } from "jotai";
import { ChevronDown, ChevronUp, List } from "lucide-react";
import React from "react";
import { Tag } from "../elements/tag";
import { Tooltip } from "../elements/tooltip";
import { GoRoutinesModel, TimelineRange } from "./goroutines-model";
import { GrTableModel } from "./grtable-model";
import { StackTrace } from "./stacktrace";

const Debug = false;

const ROW_HEIGHT = 32;

// Helper function to clean up function names by removing parens, asterisks, and .func suffixes
const cleanFuncName = (funcname: string): string => {
    let cleaned = funcname.replace(/[()*]/g, "");
    cleaned = cleaned.replace(/\.func[\d.]+$/, "");
    return cleaned;
};

// Helper function to parse and format duration strings into condensed format
const formatDurationCondensed = (durationStr: string): string | null => {
    if (!durationStr) return null;

    const patterns = [
        { regex: /^(\d+)\s*days?$/i, suffix: "d" },
        { regex: /^(\d+)\s*hours?$/i, suffix: "h" },
        { regex: /^(\d+)\s*minutes?$/i, suffix: "m" },
        { regex: /^(\d+)\s*seconds?$/i, suffix: "s" },
        { regex: /^(\d+)\s*(milliseconds?|ms)$/i, suffix: "ms" },
        { regex: /^(\d+)\s*(microseconds?|us|µs)$/i, suffix: "us" },
        { regex: /^(\d+)\s*(nanoseconds?|ns)$/i, suffix: "ns" },
    ];

    for (const pattern of patterns) {
        const match = durationStr.match(pattern.regex);
        if (match) {
            return `${match[1]}${pattern.suffix}`;
        }
    }

    return null;
};

// Helper function to format goroutine name according to the pattern [pkg].[func]:[line] or [pkg].[name]
const formatGoroutineName = (goroutine: ParsedGoRoutine): React.ReactNode => {
    const createdByFrame = goroutine.createdbyframe;
    const hasName = goroutine.name && goroutine.name.length > 0;

    if (!createdByFrame) {
        if (hasName) {
            return <span className="text-primary">{goroutine.name}</span>;
        } else {
            return <span className="text-muted">(unnamed)</span>;
        }
    }

    const pkg = createdByFrame.package.split("/").pop() || createdByFrame.package;
    const nameOrFunc = hasName ? `[${goroutine.name}]` : cleanFuncName(createdByFrame.funcname);

    return (
        <>
            {!hasName && <span className="text-secondary text-xs">{pkg}.</span>}
            <span className="text-primary">{nameOrFunc}</span>
            {!hasName && createdByFrame.linenumber && (
                <span className="text-secondary text-xs">:{createdByFrame.linenumber}</span>
            )}
        </>
    );
};

// Goroutine states: "running", "runnable", "syscall", "waiting", "IO wait", "chan send", "chan receive", "select", "sleep",
//   "sync.Mutex", "sync.RWMutex", "semacquire", "GC assist wait", "GC sweep wait", "force gc (idle)", "timer goroutine (idle)",
//   "trace reader (blocked)", "sync.WaitGroup.Wait"
const goroutineStateColors: { [state: string]: string } = {
    default: "bg-accent",
};

const columnHelper = createColumnHelper<ParsedGoRoutine>();

function cell_goid(info: any) {
    return <span className="font-mono text-sm text-secondary">{info.getValue()}</span>;
}

function cell_name(info: any, tableModel: GrTableModel, expandedRows: Set<number>) {
    const goroutine = info.row.original;
    const tags = goroutine.tags;
    const isExpanded = expandedRows.has(goroutine.goid);

    return (
        <div className="flex items-center gap-2">
            <Tooltip content="Toggle Stacktrace">
                <button
                    className={cn(
                        "flex-shrink-0 w-4 h-4 flex items-center justify-center transition-colors cursor-pointer",
                        isExpanded ? "text-primary" : "text-secondary hover:text-primary"
                    )}
                    onClick={() => tableModel.toggleRowExpanded(goroutine.goid)}
                >
                    <List className="w-3 h-3" />
                </button>
            </Tooltip>
            <div className="flex-1 flex items-center gap-2">
                <div className="text-primary">{formatGoroutineName(goroutine)}</div>
                {tags && tags.length > 0 && (
                    <div className="text-xs text-muted hover:text-primary transition-colors cursor-default">
                        {tags.map((tag: string) => `#${tag}`).join(" ")}
                    </div>
                )}
            </div>
        </div>
    );
}

function cell_primarystate(info: any, model: GoRoutinesModel) {
    const state = info.getValue();
    const goroutine = info.row.original;
    const formattedDuration = goroutine.stateduration ? formatDurationCondensed(goroutine.stateduration) : null;

    return (
        <div className="flex">
            {state ? (
                state === "inactive" ? (
                    <span className="text-muted">-</span>
                ) : (
                    <Tag
                        label={state}
                        count={formattedDuration}
                        isSelected={false}
                        variant="secondary"
                        compact={true}
                        onToggle={() => model.toggleStateFilter(state)}
                    />
                )
            ) : (
                <span className="text-muted">-</span>
            )}
        </div>
    );
}

interface GoTimelineProps {
    goroutine: ParsedGoRoutine;
    timelineRange: TimelineRange;
    model: GoRoutinesModel;
}

const GoTimeline: React.FC<GoTimelineProps> = React.memo(({ goroutine, timelineRange, model }) => {
    const grTimeSpan = useAtomValue(model.getGRTimeSpanAtom(goroutine.goid));
    const selectedTimestamp = useAtomValue(model.selectedTimestamp);
    const searchLatestMode = useAtomValue(model.searchLatestMode);
    const isAppRunning = useAtomValue(AppModel.selectedAppRunIsRunningAtom);

    const handleTimelineClick = (event: React.MouseEvent<HTMLDivElement>) => {
        const rect = event.currentTarget.getBoundingClientRect();
        const clickX = event.clientX - rect.left;
        const timelineWidth = rect.width;
        const clickPercent = clickX / timelineWidth;

        // Convert click position to timeidx
        const timeIdxRange = timelineRange.maxTimeIdx - timelineRange.minTimeIdx;
        if (timeIdxRange === 0) return;

        const clickedTimeIdx = Math.round(timelineRange.minTimeIdx + clickPercent * timeIdxRange);
        const clickedTimestamp = model.timeIdxToTimestamp(clickedTimeIdx);

        if (clickedTimestamp > 0) {
            model.setSelectedTimestampAndSearch(clickedTimestamp);
            model.focusScrubber();
        }
    };

    if (grTimeSpan?.startidx == null) {
        return <div className="h-4 bg-muted/20 rounded-sm">no startidx</div>;
    }

    // If no valid time range, show empty bar
    if (timelineRange.minTimeIdx === 0 && timelineRange.maxTimeIdx === 0) {
        return <div className="h-4 bg-muted/20 rounded-sm">empty range</div>;
    }

    const timeIdxRange = timelineRange.maxTimeIdx - timelineRange.minTimeIdx;
    if (timeIdxRange <= 0) {
        return <div className="h-4 bg-muted/20 rounded-sm">empty range 2</div>;
    }

    // Calculate the actual goroutine range in timeidx
    const grStartIdx = Math.max(grTimeSpan.startidx, timelineRange.minTimeIdx);
    const grEndIdx =
        grTimeSpan.endidx != null && grTimeSpan.endidx !== -1
            ? Math.min(grTimeSpan.endidx, timelineRange.maxTimeIdx)
            : timelineRange.maxTimeIdx;

    // Check if goroutine is still running (endidx is null or -1) AND app is running
    const isGoroutineRunning = (grTimeSpan.endidx == null || grTimeSpan.endidx === -1) && isAppRunning;

    // Calculate positions and widths using timeidx
    const startPercent = ((grStartIdx - timelineRange.minTimeIdx) / timeIdxRange) * 100;
    const widthPercent = ((grEndIdx - grStartIdx) / timeIdxRange) * 100;
    const minWidthPercent = 2;
    const finalWidthPercent = Math.max(widthPercent, minWidthPercent);

    // Calculate container width - same for all goroutines, only depends on app running state
    let containerWidthStyle: string;
    if (!isAppRunning) {
        containerWidthStyle = "100%";
    } else {
        // For running app: calculate width = actual timeline / padded timeline
        const paddedTimeIdxRange = timelineRange.paddedMaxTimeIdx - timelineRange.minTimeIdx;
        const widthPercent = (timeIdxRange / paddedTimeIdxRange) * 100;
        containerWidthStyle = `${widthPercent}%`;
    }

    // Calculate slider position marker using timeidx
    let sliderMarkerPercent: number | null = null;
    if (searchLatestMode) {
        // In search latest mode, show marker at the end
        sliderMarkerPercent = 100;
    } else if (selectedTimestamp > 0) {
        // Convert timestamp to timeidx and show marker at that position
        const selectedTimeIdx = model.timestampToTimeIdx(selectedTimestamp);
        if (selectedTimeIdx >= timelineRange.minTimeIdx && selectedTimeIdx <= timelineRange.maxTimeIdx) {
            sliderMarkerPercent = ((selectedTimeIdx - timelineRange.minTimeIdx) / timeIdxRange) * 100;
        }
    }

    // Calculate tooltip information
    const absoluteStartTime = new Date(grTimeSpan.start).toLocaleTimeString();
    const duration =
        grTimeSpan.end != null && grTimeSpan.end !== -1
            ? ((grTimeSpan.end - grTimeSpan.start) / 1000).toFixed(2)
            : "ongoing";
    const relativeStartTimeMs = grTimeSpan.start - timelineRange.startTs;
    const relativeStartTime = (relativeStartTimeMs / 1000).toFixed(3);
    const relativeStartTimeFormatted = relativeStartTimeMs >= 0 ? `+${relativeStartTime}` : relativeStartTime;

    const tooltipContent = (
        <div className="text-xs">
            <div>
                Start: {absoluteStartTime} ({relativeStartTimeFormatted}s)
            </div>
            <div>
                Duration: {duration}
                {duration !== "ongoing" ? "s" : ""}
            </div>
            {Debug && (
                <>
                    <div className="text-muted mt-1">
                        Debug: startidx={grTimeSpan.startidx}, endidx={grTimeSpan.endidx}
                    </div>
                    <div className="text-muted mt-1">
                        Debug: start={grTimeSpan.start - timelineRange.startTs}, end=
                        {grTimeSpan.end - timelineRange.startTs}
                    </div>
                </>
            )}
        </div>
    );

    return (
        <div
            className="relative h-4 bg-muted/20 rounded-sm overflow-visible cursor-pointer"
            style={{ width: containerWidthStyle }}
            onClick={handleTimelineClick}
        >
            <Tooltip content={tooltipContent}>
                <div
                    className="absolute h-full bg-accent rounded-sm cursor-pointer"
                    style={{
                        left: `${startPercent}%`,
                        width: `${finalWidthPercent}%`,
                    }}
                />
            </Tooltip>
            {sliderMarkerPercent !== null && (
                <div
                    className="absolute w-[2px] bg-black pointer-events-none z-[1.5] rounded-lg"
                    style={{
                        left: `${sliderMarkerPercent}%`,
                        top: "-2px",
                        height: "calc(100% + 4px)",
                    }}
                />
            )}
        </div>
    );
});

GoTimeline.displayName = "GoTimeline";

function cell_timeline(info: any, timelineRange: TimelineRange, model: GoRoutinesModel) {
    const goroutine: ParsedGoRoutine = info.row.original;
    return <GoTimeline goroutine={goroutine} timelineRange={timelineRange} model={model} />;
}

interface GoRoutinesTableProps {
    sortedGoroutines: ParsedGoRoutine[];
    tableModel: GrTableModel;
    model: GoRoutinesModel;
}

export const GoRoutinesTable: React.FC<GoRoutinesTableProps> = ({ sortedGoroutines, tableModel, model }) => {
    const containerSize = useAtomValue(tableModel.containerSize);
    const columns = useAtomValue(tableModel.columns);
    const simpleMode = useAtomValue(model.effectiveSimpleStacktraceMode);
    const expandedRows = useAtomValue(tableModel.expandedRows);
    const timelineRange = useAtomValue(model.timelineRangeAtom);

    const getColumnGrow = (columnId: string): number => {
        const column = columns.find((col) => col.id === columnId);
        return column?.grow || 0;
    };

    // Sort functions for each column
    const sortByName = (rowA: any, rowB: any): number => {
        const a = rowA.original as ParsedGoRoutine;
        const b = rowB.original as ParsedGoRoutine;

        const getNameForSort = (gr: ParsedGoRoutine): string => {
            if (gr.name && gr.name.length > 0) {
                return gr.name.toLowerCase();
            }
            if (gr.createdbyframe) {
                const pkg = gr.createdbyframe.package.split("/").pop() || gr.createdbyframe.package;
                const func = cleanFuncName(gr.createdbyframe.funcname);
                return `${pkg}.${func}`.toLowerCase();
            }
            return "";
        };

        return getNameForSort(a).localeCompare(getNameForSort(b));
    };

    const sortByState = (rowA: any, rowB: any): number => {
        const a = rowA.original as ParsedGoRoutine;
        const b = rowB.original as ParsedGoRoutine;

        const aState = a.primarystate || "";
        const bState = b.primarystate || "";

        // "inactive" should sort to the bottom (be the "largest" value)
        if (aState === "inactive" && bState !== "inactive") return 1;
        if (bState === "inactive" && aState !== "inactive") return -1;
        if (aState === "inactive" && bState === "inactive") return a.goid - b.goid;

        // For other states, sort alphabetically with sub-sort by goid
        const comparison = aState.localeCompare(bState);
        return comparison === 0 ? a.goid - b.goid : comparison;
    };

    const sortByTimeline = (rowA: any, rowB: any): number => {
        const a = rowA.original as ParsedGoRoutine;
        const b = rowB.original as ParsedGoRoutine;

        const store = getDefaultStore();
        const aSpanAtom = model.getGRTimeSpanAtom(a.goid);
        const bSpanAtom = model.getGRTimeSpanAtom(b.goid);
        const aSpan = store.get(aSpanAtom);
        const bSpan = store.get(bSpanAtom);

        // Handle cases where timespan might not exist
        if (!aSpan && !bSpan) return a.goid - b.goid; // Sub-sort by goid
        if (!aSpan) return 1;
        if (!bSpan) return -1;

        // Sort by start time (startidx)
        const aStart = aSpan.startidx ?? 0;
        const bStart = bSpan.startidx ?? 0;

        // If start times are equal, sub-sort by goid
        if (aStart === bStart) {
            return a.goid - b.goid;
        }

        return aStart - bStart;
    };

    const tableColumns = [
        columnHelper.accessor("goid", {
            header: "GoID",
            cell: cell_goid,
            size: tableModel.getColumnWidth("goid"),
            enableResizing: true,
            enableSorting: true,
        }),
        columnHelper.accessor((row) => row, {
            id: "name",
            header: "Name",
            cell: (info) => cell_name(info, tableModel, expandedRows),
            size: tableModel.getColumnWidth("name"),
            enableResizing: true,
            enableSorting: true,
            sortingFn: sortByName,
        }),
        columnHelper.accessor("primarystate", {
            header: "State",
            cell: (info) => cell_primarystate(info, model),
            size: tableModel.getColumnWidth("state"),
            enableResizing: true,
            enableSorting: true,
            sortingFn: sortByState,
        }),
        columnHelper.accessor((row) => row, {
            id: "timeline",
            header: "Timeline",
            cell: (info) => cell_timeline(info, timelineRange, model),
            size: tableModel.getColumnWidth("timeline"),
            enableResizing: true,
            enableSorting: true,
            sortingFn: sortByTimeline,
        }),
    ];

    const table = useReactTable({
        data: sortedGoroutines,
        columns: tableColumns,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        columnResizeMode: "onChange",
        enableColumnResizing: true,
        enableSorting: true,
        sortDescFirst: false,
        initialState: {
            sorting: [
                {
                    id: "timeline",
                    desc: false,
                },
            ],
        },
        defaultColumn: {
            minSize: 50,
        },
    });

    return (
        <>
            <div className="w-full">
                <div className="sticky top-0 bg-panel border-b border-border z-2">
                    {table.getHeaderGroups().map((headerGroup) => (
                        <div key={headerGroup.id} className="flex">
                            {headerGroup.headers.map((header) => (
                                <div
                                    key={header.id}
                                    className={cn(
                                        "text-left p-3 text-sm font-medium text-secondary flex items-center gap-1",
                                        getColumnGrow(header.id) > 0 ? "flex-grow" : "",
                                        header.column.getCanSort() ? "cursor-pointer hover:text-primary" : ""
                                    )}
                                    style={getColumnGrow(header.id) > 0 ? {} : { width: header.getSize() }}
                                    onClick={header.column.getToggleSortingHandler()}
                                >
                                    {header.isPlaceholder
                                        ? null
                                        : flexRender(header.column.columnDef.header, header.getContext())}
                                    {header.column.getCanSort() && (
                                        <div className="flex flex-col">
                                            {header.column.getIsSorted() === "asc" ? (
                                                <ChevronUp className="w-3 h-3" />
                                            ) : header.column.getIsSorted() === "desc" ? (
                                                <ChevronDown className="w-3 h-3" />
                                            ) : (
                                                <div className="w-3 h-3 opacity-30">
                                                    <ChevronUp className="w-3 h-3 absolute" />
                                                    <ChevronDown className="w-3 h-3 absolute translate-y-1" />
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    ))}
                </div>

                <div>
                    {table.getRowModel().rows.map((row) => {
                        const goroutine = row.original;
                        const isExpanded = expandedRows.has(goroutine.goid);

                        return (
                            <React.Fragment key={row.id}>
                                <div
                                    key="maindiv"
                                    className="flex border-b border-border hover:bg-muted/5 transition-colors"
                                    style={{ height: ROW_HEIGHT }}
                                >
                                    {row.getVisibleCells().map((cell) => (
                                        <div
                                            key={cell.id}
                                            className={cn(
                                                "px-3 text-sm flex items-center",
                                                getColumnGrow(cell.column.id) > 0 ? "flex-grow" : ""
                                            )}
                                            style={
                                                getColumnGrow(cell.column.id) > 0
                                                    ? {}
                                                    : { width: cell.column.getSize() }
                                            }
                                        >
                                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                        </div>
                                    ))}
                                </div>
                                {isExpanded && (
                                    <div key="stacktracediv" className="border-b border-border bg-panel/50">
                                        <div className="px-3 py-2">
                                            {goroutine.active ? (
                                                <StackTrace
                                                    goroutine={goroutine}
                                                    model={model}
                                                    simpleMode={simpleMode}
                                                />
                                            ) : (
                                                <div className="text-secondary italic text-sm ml-1.5">
                                                    Goroutine was inactive at this time, no stack trace available
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </React.Fragment>
                        );
                    })}
                </div>
            </div>
        </>
    );
};
