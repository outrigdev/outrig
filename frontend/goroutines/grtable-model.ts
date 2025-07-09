// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

// Column configuration type
export type ColumnConfig = {
    id: string;
    width: number;
    grow: number; // 0 for fixed width, >0 for flex-grow
};

// Container size type
export type ContainerSize = {
    width: number;
    height: number;
};

// Default column configurations
const DEFAULT_COLUMNS: ColumnConfig[] = [
    {
        id: "goid",
        width: 80,
        grow: 0,
    },
    {
        id: "name",
        width: 250,
        grow: 0,
    },
    {
        id: "state",
        width: 150,
        grow: 0,
    },
    {
        id: "timeline",
        width: 200,
        grow: 1, // This column will grow to fill remaining space
    },
];

class GrTableModel {
    // Column configurations
    columns: PrimitiveAtom<ColumnConfig[]> = atom(DEFAULT_COLUMNS);

    // Container size tracking
    containerSize: PrimitiveAtom<ContainerSize> = atom({ width: 0, height: 0 });

    // Resize observer reference
    private resizeObserver: ResizeObserver | null = null;
    private containerElement: HTMLElement | null = null;

    constructor() {
        // Initialize resize observer
        this.resizeObserver = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const { width, height } = entry.contentRect;
                this.updateContainerSize({ width, height });
            }
        });
    }

    // Set up resize observation for a container element
    observeContainer(element: HTMLElement | null) {
        if (this.containerElement && this.resizeObserver) {
            this.resizeObserver.unobserve(this.containerElement);
        }

        this.containerElement = element;

        if (element && this.resizeObserver) {
            this.resizeObserver.observe(element);
            // Get initial size
            const rect = element.getBoundingClientRect();
            this.updateContainerSize({ width: rect.width, height: rect.height });
        }
    }

    // Update container size
    private updateContainerSize(size: ContainerSize) {
        const store = getDefaultStore();
        store.set(this.containerSize, size);
    }

    // Update column width
    updateColumnWidth(columnId: string, width: number) {
        const store = getDefaultStore();
        const currentColumns = store.get(this.columns);
        const updatedColumns = currentColumns.map((col: ColumnConfig) => {
            if (col.id === columnId) {
                return { ...col, width: Math.max(50, width) }; // Minimum 50px width
            }
            return col;
        });

        store.set(this.columns, updatedColumns);
    }

    // Get total width of fixed columns (grow = 0)
    getFixedColumnsWidth(): number {
        const store = getDefaultStore();
        const columns = store.get(this.columns);
        return columns
            .filter((col: ColumnConfig) => col.grow === 0)
            .reduce((total: number, col: ColumnConfig) => total + col.width, 0);
    }

    // Get calculated table width based on container and column sizes
    getTableWidth(): number {
        const store = getDefaultStore();
        const containerSize = store.get(this.containerSize);
        const fixedWidth = this.getFixedColumnsWidth();
        const columns = store.get(this.columns);
        const growColumns = columns.filter((col: ColumnConfig) => col.grow > 0);

        if (growColumns.length === 0) {
            // No growing columns, use fixed width
            return fixedWidth;
        }

        // Calculate minimum width for growing columns
        const minGrowWidth = growColumns.reduce((total: number, col: ColumnConfig) => total + col.width, 0);
        const minRequiredWidth = fixedWidth + minGrowWidth;

        // Return the larger of container width or minimum required width
        return Math.max(containerSize.width, minRequiredWidth);
    }

    // Get column width (for growing columns, calculate based on available space)
    getColumnWidth(columnId: string): number {
        const store = getDefaultStore();
        const columns = store.get(this.columns);
        const column = columns.find((col: ColumnConfig) => col.id === columnId);

        if (!column) return 100;

        if (column.grow === 0) {
            return column.width;
        }

        // For growing columns, calculate based on available space
        const tableWidth = this.getTableWidth();
        const fixedWidth = this.getFixedColumnsWidth();
        const availableWidth = tableWidth - fixedWidth;
        const growColumns = columns.filter((col: ColumnConfig) => col.grow > 0);
        const totalGrow = growColumns.reduce((total: number, col: ColumnConfig) => total + col.grow, 0);

        if (totalGrow === 0) return column.width;

        const growWidth = (availableWidth * column.grow) / totalGrow;
        return Math.max(column.width, growWidth);
    }

    // Reset columns to default
    resetColumns() {
        const store = getDefaultStore();
        store.set(this.columns, [...DEFAULT_COLUMNS]);
    }

    // Clean up resources
    dispose() {
        if (this.resizeObserver) {
            this.resizeObserver.disconnect();
            this.resizeObserver = null;
        }
        this.containerElement = null;
    }
}

export { GrTableModel };
