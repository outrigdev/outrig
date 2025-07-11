// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useAtomValue } from "jotai";
import { List } from "lucide-react";
import React from "react";
import { Tag } from "../elements/tag";
import { Tooltip } from "../elements/tooltip";
import { GoRoutinesModel } from "./goroutines-model";
import { GrTableModel } from "./grtable-model";
import { StackTrace } from "./stacktrace";

const ROW_HEIGHT = 45;

// Helper function to clean up function names by removing parens, asterisks, and .func suffixes
const cleanFuncName = (funcname: string): string => {
    let cleaned = funcname.replace(/[()*]/g, "");
    cleaned = cleaned.replace(/\.func[\d.]+$/, "");
    return cleaned;
};

// Helper function to format goroutine name according to the pattern [pkg].[func]#[csnum] or [pkg].[name]#[csnum]
const formatGoroutineName = (goroutine: ParsedGoRoutine): React.ReactNode => {
    const createdByFrame = goroutine.createdbyframe;

    if (!createdByFrame) {
        if (goroutine.name) {
            return <span className="text-primary">{goroutine.name}</span>;
        } else {
            return <span className="text-muted">(unnamed)</span>;
        }
    }

    const pkg = createdByFrame.package.split("/").pop() || createdByFrame.package;
    const nameOrFunc = goroutine.name ? `[${goroutine.name}]` : cleanFuncName(createdByFrame.funcname);

    return (
        <>
            {!goroutine.name && <span className="text-secondary">{pkg}.</span>}
            <span className="text-primary">{nameOrFunc}</span>
            {goroutine.csnum && goroutine.csnum !== 0 && <span className="text-secondary">#{goroutine.csnum}</span>}
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
        <div className="flex items-start gap-2">
            <Tooltip content="Toggle Stacktrace">
                <button
                    className={cn(
                        "flex-shrink-0 w-4 h-4 flex items-center justify-center transition-colors mt-0.5 cursor-pointer",
                        isExpanded ? "text-primary" : "text-secondary hover:text-primary"
                    )}
                    onClick={() => tableModel.toggleRowExpanded(goroutine.goid)}
                >
                    <List className="w-3 h-3" />
                </button>
            </Tooltip>
            <div className="flex-1">
                <div className="text-primary">{formatGoroutineName(goroutine)}</div>
                {tags && tags.length > 0 && (
                    <div className="text-xs text-muted hover:text-primary mt-0.5 transition-colors cursor-default">
                        {tags.map((tag: string) => `#${tag}`).join(" ")}
                    </div>
                )}
            </div>
        </div>
    );
}

function cell_primarystate(info: any) {
    const state = info.getValue();
    const goroutine = info.row.original;
    return (
        <div className="flex">
            {state ? (
                <Tag label={state} isSelected={false} variant="secondary" />
            ) : (
                <span className="text-muted">-</span>
            )}
        </div>
    );
}

interface GoTimelineProps {
    goroutine: ParsedGoRoutine;
    timelineRange: { startTime: number; endTime: number };
    model: GoRoutinesModel;
}

const GoTimeline: React.FC<GoTimelineProps> = React.memo(({ goroutine, timelineRange, model }) => {
    const grTimeSpan = useAtomValue(model.getGRTimeSpanAtom(goroutine.goid));
    const selectedTimestamp = useAtomValue(model.selectedTimestamp);
    const searchLatestMode = useAtomValue(model.searchLatestMode);

    if (!grTimeSpan?.start) {
        return <div className="h-4 bg-muted/20 rounded-sm"></div>;
    }

    const { startTime, endTime } = timelineRange;

    // If no valid time range, show empty bar
    if (startTime === 0 && endTime === 0) {
        return <div className="h-4 bg-muted/20 rounded-sm"></div>;
    }

    const totalDuration = endTime - startTime;
    if (totalDuration <= 0) {
        return <div className="h-4 bg-muted/20 rounded-sm"></div>;
    }

    // Calculate positions as percentages
    const grStartTime = Math.max(grTimeSpan.start, startTime);
    // If end is 0 or null, it spans to the end of the range
    const grEndTime = grTimeSpan.end && grTimeSpan.end > 0 ? Math.min(grTimeSpan.end, endTime) : endTime;

    const startPercent = ((grStartTime - startTime) / totalDuration) * 100;
    const widthPercent = ((grEndTime - grStartTime) / totalDuration) * 100;

    // Ensure minimum 2% width for visibility
    const minWidthPercent = 2;
    const finalWidthPercent = Math.max(widthPercent, minWidthPercent);

    // Calculate slider position marker
    let sliderMarkerPercent: number | null = null;
    if (!searchLatestMode && selectedTimestamp > 0) {
        // Only show marker if we have a specific timestamp selected (not in latest mode)
        if (selectedTimestamp >= startTime && selectedTimestamp <= endTime) {
            sliderMarkerPercent = ((selectedTimestamp - startTime) / totalDuration) * 100;
        }
    }

    // Calculate tooltip information
    const absoluteStartTime = new Date(grTimeSpan.start).toLocaleTimeString();
    const relativeStartTime = ((grTimeSpan.start - startTime) / 1000).toFixed(2);
    const duration =
        grTimeSpan.end && grTimeSpan.end > 0 ? ((grTimeSpan.end - grTimeSpan.start) / 1000).toFixed(2) : "ongoing";

    const tooltipContent = (
        <div className="text-xs">
            <div>
                Start: {absoluteStartTime} (+{relativeStartTime}s)
            </div>
            <div>
                Duration: {duration}
                {duration !== "ongoing" ? "s" : ""}
            </div>
        </div>
    );

    return (
        <div className="relative h-4 bg-muted/20 rounded-sm overflow-visible w-full">
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

function cell_timeline(info: any, timelineRange: { startTime: number; endTime: number }, model: GoRoutinesModel) {
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
    const fullTimeSpan = useAtomValue(model.fullTimeSpan);

    const getColumnGrow = (columnId: string): number => {
        const column = columns.find((col) => col.id === columnId);
        return column?.grow || 0;
    };

    // Calculate timeline range once for all rows using fullTimeSpan
    const timelineRange = React.useMemo(() => {
        if (!fullTimeSpan?.start) {
            return { startTime: 0, endTime: 0 };
        }

        const startTime = fullTimeSpan.start;
        const endTime = fullTimeSpan.end && fullTimeSpan.end > 0 ? fullTimeSpan.end : Date.now();

        // If starttime is more than 600s before endtime, set starttime to endtime - 600s
        const maxStartTime = endTime - 600 * 1000;
        const effectiveStartTime = Math.max(startTime, maxStartTime);

        return {
            startTime: effectiveStartTime,
            endTime,
        };
    }, [fullTimeSpan]);

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
            cell: cell_primarystate,
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
