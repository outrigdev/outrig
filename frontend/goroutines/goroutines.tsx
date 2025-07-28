// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { DroppedGoroutinesIndicator } from "./goroutines-comps";
import { GoRoutinesFilters } from "./goroutines-filters";
import { GoRoutinesModel } from "./goroutines-model";
import { GoRoutinesTable } from "./goroutines-table";
import { GrTableModel } from "./grtable-model";

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
    const containerSize = useAtomValue(tableModel.containerSize);
    const containerRef = useRef<HTMLDivElement>(null);
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
        model.setContentRef(containerRef);
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

    return (
        <div ref={containerRef} className="w-full h-full overflow-auto flex-1 relative">
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
            ) : containerSize.width > 0 ? (
                <GoRoutinesTable tableModel={tableModel} model={model} />
            ) : null}
            <DroppedGoroutinesIndicator model={model} />
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
