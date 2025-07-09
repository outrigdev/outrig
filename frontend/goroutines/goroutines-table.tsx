// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import React from "react";
import { Tag } from "../elements/tag";
import { GrTableModel } from "./grtable-model";

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
            return <span className="text-muted">-</span>;
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

const columnHelper = createColumnHelper<ParsedGoRoutine>();

interface GoRoutinesTableProps {
    sortedGoroutines: ParsedGoRoutine[];
    tableModel: GrTableModel;
}

export const GoRoutinesTable: React.FC<GoRoutinesTableProps> = ({ sortedGoroutines, tableModel }) => {
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
        <div className="w-full">
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

            <div>
                {table.getRowModel().rows.map((row) => (
                    <div key={row.id} className="flex border-b border-border hover:bg-muted/5 transition-colors">
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
    );
};
