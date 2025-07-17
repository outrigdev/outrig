// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { cn } from "@/util/util";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useAtomValue } from "jotai";
import { List } from "lucide-react";
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
    return (
        <div className="flex">
            {state ? (
                state === "inactive" ? (
                    <span className="text-muted">-</span>
                ) : (
                    <Tag
                        label={state}
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
            <div>Start: {absoluteStartTime} ({relativeStartTimeFormatted}s)</div>
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

    const tableColumns = [
        columnHelper.accessor("goid", {
            header: "GoID",
            cell: cell_goid,
            size: tableModel.getColumnWidth("goid"),
            enableResizing: true,
        }),
        columnHelper.display({
            id: "name",
            header: "Name",
            cell: (info) => cell_name(info, tableModel, expandedRows),
            size: tableModel.getColumnWidth("name"),
            enableResizing: true,
        }),
        columnHelper.accessor("primarystate", {
            header: "State",
            cell: (info) => cell_primarystate(info, model),
            size: tableModel.getColumnWidth("state"),
            enableResizing: true,
        }),
        columnHelper.display({
            id: "timeline",
            header: "Timeline",
            cell: (info) => cell_timeline(info, timelineRange, model),
            size: tableModel.getColumnWidth("timeline"),
            enableResizing: true,
        }),
    ];

    const table = useReactTable({
        data: sortedGoroutines,
        columns: tableColumns,
        getCoreRowModel: getCoreRowModel(),
        columnResizeMode: "onChange",
        enableColumnResizing: true,
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
                                        "text-left p-3 text-sm font-medium text-secondary",
                                        getColumnGrow(header.id) > 0 ? "flex-grow" : ""
                                    )}
                                    style={getColumnGrow(header.id) > 0 ? {} : { width: header.getSize() }}
                                >
                                    {header.isPlaceholder
                                        ? null
                                        : flexRender(header.column.columnDef.header, header.getContext())}
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
                                            <StackTrace goroutine={goroutine} model={model} simpleMode={simpleMode} />
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
