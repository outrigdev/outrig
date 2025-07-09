// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { cn } from "@/util/util";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { Tag } from "../elements/tag";
import { GoRoutinesFilters } from "./goroutines-filters";
import { GoRoutinesModel } from "./goroutines-model";
import { GrTableModel } from "./grtable-model";

// Helper function to clean up function names by removing parens, asterisks, and .func suffixes
const cleanFuncName = (funcname: string): string => {
    // Remove parentheses and asterisks
    let cleaned = funcname.replace(/[()*]/g, "");

    // Remove .func[\d.]+ suffixes (like .func5 or .func5.2)
    cleaned = cleaned.replace(/\.func[\d.]+$/, "");

    return cleaned;
};

// Helper function to format goroutine name according to the pattern [pkg].[func]#[csnum] or [pkg].[name]#[csnum]
const formatGoroutineName = (goroutine: ParsedGoRoutine): React.ReactNode => {
    // Use the createdbyframe to extract package and function info
    const createdByFrame = goroutine.createdbyframe;

    if (!createdByFrame) {
        // Fallback to original display if no createdbyframe
        if (goroutine.name) {
            return <span className="text-primary">{goroutine.name}</span>;
        } else {
            return <span className="text-muted">-</span>;
        }
    }

    // Extract just the last part of the package path (after the last slash)
    const pkg = createdByFrame.package.split("/").pop() || createdByFrame.package;

    // Determine what to show after the package
    const nameOrFunc = goroutine.name ? `[${goroutine.name}]` : cleanFuncName(createdByFrame.funcname);

    return (
        <>
            {!goroutine.name && <span className="text-secondary">{pkg}.</span>}
            <span className="text-primary">{nameOrFunc}</span>
            {goroutine.csnum && goroutine.csnum !== 0 && <span className="text-secondary">#{goroutine.csnum}</span>}
        </>
    );
};

// Create column helper for type safety
const columnHelper = createColumnHelper<ParsedGoRoutine>();

// Content component that displays the goroutines table
interface GoRoutinesContentProps {
    model: GoRoutinesModel;
    tableModel: GrTableModel;
}

const GoRoutinesContent: React.FC<GoRoutinesContentProps> = ({ model, tableModel }) => {
    const sortedGoroutines = useAtomValue(model.sortedGoRoutines);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const search = useAtomValue(model.searchTerm);
    const showAll = useAtomValue(model.showAll);
    const tableModelColumns = useAtomValue(tableModel.columns);
    const containerSize = useAtomValue(tableModel.containerSize);
    const containerRef = useRef<HTMLDivElement>(null);
    const contentRef = useRef<HTMLDivElement>(null);
    const [showEmptyMessage, setShowEmptyMessage] = useState(false);

    // Set up resize observation for the container
    useEffect(() => {
        tableModel.observeContainer(containerRef.current);
        return () => {
            tableModel.dispose();
        };
    }, [tableModel]);

    // Set the content ref in the model when it changes
    useEffect(() => {
        model.setContentRef(contentRef);
    }, [model]);

    // Set a timeout to show empty message after component mounts or when goroutines change
    useEffect(() => {
        if (sortedGoroutines.length === 0 && !isRefreshing) {
            const timer = setTimeout(() => {
                setShowEmptyMessage(true);
            }, EmptyMessageDelayMs);

            return () => clearTimeout(timer);
        } else {
            setShowEmptyMessage(false);
        }
    }, [sortedGoroutines.length, isRefreshing]);

    // Define table columns using the table model configuration
    const tableColumns = [
        columnHelper.accessor("goid", {
            header: "GoID",
            cell: (info) => <span className="font-mono text-sm text-secondary">{info.getValue()}</span>,
            size: tableModel.getColumnWidth("goid"),
            enableResizing: true,
        }),
        columnHelper.display({
            id: "name",
            header: "Name",
            cell: (info) => <div className="text-primary">{formatGoroutineName(info.row.original)}</div>,
            size: tableModel.getColumnWidth("name"),
            enableResizing: true,
        }),
        columnHelper.accessor("primarystate", {
            header: "State",
            cell: (info) => {
                const state = info.getValue();
                const goroutine = info.row.original;
                return (
                    <div className="flex">
                        {state ? (
                            <Tag
                                label={goroutine.stateduration ? `${state} (${goroutine.stateduration})` : state}
                                isSelected={false}
                                variant="secondary"
                            />
                        ) : (
                            <span className="text-muted">-</span>
                        )}
                    </div>
                );
            },
            size: tableModel.getColumnWidth("state"),
            enableResizing: true,
        }),
        columnHelper.display({
            id: "timeline",
            header: "Timeline",
            cell: (info) => <div className="text-secondary">Timeline placeholder</div>,
            size: tableModel.getColumnWidth("timeline"),
            enableResizing: true,
        }),
    ];

    // Create table instance
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
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing goroutines...</span>
                    </div>
                </div>
            ) : sortedGoroutines.length === 0 && showEmptyMessage ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search || !showAll ? "no goroutines match the filter" : "no goroutines found"}
                </div>
            ) : (
                <div className="w-full">
                    {/* Header */}
                    <div className="sticky top-0 bg-background border-b border-border">
                        {table.getHeaderGroups().map((headerGroup) => (
                            <div key={headerGroup.id} className="flex">
                                {headerGroup.headers.map((header) => (
                                    <div
                                        key={header.id}
                                        className={cn(
                                            "text-left p-3 text-sm font-medium text-secondary",
                                            header.id === "timeline" ? "flex-grow" : ""
                                        )}
                                        style={header.id === "timeline" ? {} : { width: header.getSize() }}
                                    >
                                        {header.isPlaceholder
                                            ? null
                                            : flexRender(header.column.columnDef.header, header.getContext())}
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>

                    {/* Body */}
                    <div>
                        {table.getRowModel().rows.map((row) => (
                            <div
                                key={row.id}
                                className="flex border-b border-border hover:bg-muted/5 transition-colors"
                            >
                                {row.getVisibleCells().map((cell) => (
                                    <div
                                        key={cell.id}
                                        className={cn("p-3 text-sm", cell.column.id === "timeline" ? "flex-grow" : "")}
                                        style={cell.column.id === "timeline" ? {} : { width: cell.column.getSize() }}
                                    >
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
};

// Main goroutines component that composes the sub-components
interface GoRoutinesProps {
    appRunId: string;
}

export const GoRoutines: React.FC<GoRoutinesProps> = ({ appRunId }) => {
    const model = useOutrigModel(GoRoutinesModel, appRunId);
    const [tableModel] = useState(() => new GrTableModel());

    // Clean up table model on unmount
    useEffect(() => {
        return () => {
            tableModel.dispose();
        };
    }, [tableModel]);

    if (!model) {
        return null;
    }

    return (
        <div className="w-full h-full flex flex-col">
            <GoRoutinesFilters model={model} />
            <GoRoutinesContent model={model} tableModel={tableModel} />
        </div>
    );
};
