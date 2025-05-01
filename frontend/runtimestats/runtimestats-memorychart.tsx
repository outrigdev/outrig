// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { useRef, useState } from "react";
import {
    FloatingPortal,
    autoUpdate,
    flip,
    offset,
    shift,
    useFloating,
    useHover,
    useInteractions,
} from "@floating-ui/react";
import { RuntimeStatsTooltip } from "./tooltip";

// Memory chart metadata
interface MemoryChartSegmentMetadata {
    id: string;
    label: string;
    color: string;
    valueFn: (memStats: MemoryStatsInfo) => number;
    percentFn: (memStats: MemoryStatsInfo) => number;
    desc: string;
}

const memoryChartMetadata: MemoryChartSegmentMetadata[] = [
    {
        id: "heap",
        label: "Heap In Use",
        color: "bg-blue-600",
        valueFn: (memStats) => memStats.heapinuse / (1024 * 1024),
        percentFn: (memStats) => (memStats.heapinuse / memStats.sys) * 100,
        desc: "Memory currently allocated and in use by the Go heap for storing application data.",
    },
    {
        id: "stack",
        label: "Stack",
        color: "bg-green-600",
        valueFn: (memStats) => memStats.stackinuse / (1024 * 1024),
        percentFn: (memStats) => (memStats.stackinuse / memStats.sys) * 100,
        desc: "Memory used by goroutine stacks. Each goroutine has its own stack that grows and shrinks as needed.",
    },
    {
        id: "other",
        label: "Other",
        color: "bg-yellow-600",
        valueFn: (memStats) =>
            (memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / (1024 * 1024),
        percentFn: (memStats) =>
            ((memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / memStats.sys) * 100,
        desc: "Other memory used by the Go runtime, including memory spans, mcache, garbage collector, and other system memory.",
    },
    {
        id: "idle",
        label: "Idle",
        color: "bg-gray-400",
        valueFn: (memStats) => memStats.heapidle / (1024 * 1024),
        percentFn: (memStats) => (memStats.heapidle / memStats.sys) * 100,
        desc: "Memory in the heap that is not currently in use but has been allocated from the OS. This memory can be reused by the application without requesting more from the OS.",
    },
];

// Helper function to get detailed memory breakdown for the "other" category
function getDetailedOtherMemoryBreakdown(memStats: MemoryStatsInfo): string {
    return `Memory spans: ${(memStats.mspaninuse / (1024 * 1024)).toFixed(2)} MB
MCache: ${(memStats.mcacheinuse / (1024 * 1024)).toFixed(2)} MB
GC: ${(memStats.gcsys / (1024 * 1024)).toFixed(2)} MB
Other: ${(memStats.othersys / (1024 * 1024)).toFixed(2)} MB`;
}

// Memory usage chart component
interface MemoryUsageChartProps {
    memStats: MemoryStatsInfo;
}

export const MemoryUsageChart: React.FC<MemoryUsageChartProps> = ({ memStats }) => {
    // Calculate values and percentages using metadata
    const segments = memoryChartMetadata.map((segment) => ({
        id: segment.id,
        label: segment.label,
        color: segment.color,
        valueMB: segment.valueFn(memStats).toFixed(2),
        percent: segment.percentFn(memStats),
        desc: segment.desc,
    }));

    // Create tooltip content for each segment
    const createTooltipContent = (segment: (typeof segments)[0]) => (
        <div>
            <div className="font-medium mb-1">{segment.label}</div>
            <div className="text-secondary mb-2">
                {segment.valueMB} MB ({segment.percent.toFixed(1)}% of total)
            </div>
            <div className="text-xs">
                {segment.desc}
                {segment.id === "other" && (
                    <div className="mt-1">
                        {getDetailedOtherMemoryBreakdown(memStats)
                            .split("\n")
                            .map((line, i) => (
                                <div key={i}>{line}</div>
                            ))}
                    </div>
                )}
            </div>
        </div>
    );

    // State to track which segment is being hovered
    const [hoveredSegment, setHoveredSegment] = useState<string | null>(null);
    const [open, setOpen] = useState(false);

    // References for the chart and hovered segment
    const chartRef = useRef<HTMLDivElement>(null);
    const segmentRefs = useRef<Record<string, HTMLDivElement | null>>({});

    // Set up floating UI
    const { refs, floatingStyles, context } = useFloating({
        open,
        onOpenChange: setOpen,
        placement: "bottom",
        middleware: [offset(10), flip(), shift()],
        whileElementsMounted: autoUpdate,
    });

    // Set up hover interaction
    const hover = useHover(context);
    const { getFloatingProps } = useInteractions([hover]);

    // Handle segment hover
    const handleSegmentHover = (segmentId: string, element: HTMLDivElement) => {
        setHoveredSegment(segmentId);
        setOpen(true);
        refs.setReference(element);
    };

    // Handle mouse leave
    const handleMouseLeave = () => {
        setHoveredSegment(null);
        setOpen(false);
    };

    return (
        <div>
            <div
                className="relative flex h-6 w-full rounded-md overflow-hidden mb-2"
                ref={chartRef}
                onMouseLeave={handleMouseLeave}
            >
                {segments.map((segment) => (
                    <div
                        key={segment.id}
                        ref={(el) => {
                            if (el) {
                                segmentRefs.current[segment.id] = el;
                            }
                        }}
                        className={`${segment.color} h-full`}
                        style={{ width: `${segment.percent}%` }}
                        onMouseEnter={() => {
                            const element = segmentRefs.current[segment.id];
                            if (element) {
                                handleSegmentHover(segment.id, element);
                            }
                        }}
                    />
                ))}

                {open && hoveredSegment && (
                    <FloatingPortal>
                        <div
                            ref={refs.setFloating}
                            style={floatingStyles}
                            {...getFloatingProps()}
                            className="bg-panel border border-border rounded-md px-3 py-2 text-sm text-primary shadow-md z-50 max-w-xs"
                        >
                            {createTooltipContent(segments.find((s) => s.id === hoveredSegment)!)}
                        </div>
                    </FloatingPortal>
                )}
            </div>
            <div className="flex flex-wrap text-xs gap-3 mb-2">
                {segments.map((segment) => (
                    <RuntimeStatsTooltip key={segment.id} content={createTooltipContent(segment)}>
                        <div className="flex items-center">
                            <div className={`w-3 h-3 ${segment.color} mr-1 rounded-sm`}></div>
                            <span className="text-primary">
                                {segment.label}: {segment.valueMB} MB
                            </span>
                        </div>
                    </RuntimeStatsTooltip>
                ))}
            </div>
            <div className="text-xs text-secondary mt-2">
                Total Process Memory: {(memStats.sys / (1024 * 1024)).toFixed(2)} MB
            </div>
        </div>
    );
};
