// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { formatMemorySize, FormattedMemory } from "@/util/util";
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
import React, { useRef, useState } from "react";
import { RuntimeStatsTooltip } from "./tooltip";

// Memory usage chart component
interface MemoryUsageChartProps {
    memStats: MemoryStatsInfo;
}

// Segment type definition
interface MemorySegment {
    id: string;
    label: string;
    color: string;
    mem: FormattedMemory;  // The formatted memory object
    percent: number;
    desc: string;
}

// Helper function to get detailed memory breakdown for the runtime category
function getDetailedRuntimeMemoryBreakdown(memStats: MemoryStatsInfo): React.ReactNode {
    const spans = formatMemorySize(memStats.mspaninuse);
    const mcache = formatMemorySize(memStats.mcacheinuse);
    const gc = formatMemorySize(memStats.gcsys);
    const other = formatMemorySize(memStats.othersys);
    
    return (
        <div className="mt-1">
            <div>
                Memory spans: {spans.memstr} {spans.memunit}
            </div>
            <div>
                MCache: {mcache.memstr} {mcache.memunit}
            </div>
            <div>
                GC: {gc.memstr} {gc.memunit}
            </div>
            <div>
                Other: {other.memstr} {other.memunit}
            </div>
        </div>
    );
}

export const MemoryUsageChart: React.FC<MemoryUsageChartProps> = ({ memStats }) => {
    // Define segments directly
    const runtimeMem = memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys;
    const stackMem = memStats.stackinuse;
    const heapMem = memStats.heapinuse;
    const idleMem = memStats.heapidle;
    const totalMem = memStats.sys;

    // Calculate percentages
    const runtimePercent = (runtimeMem / totalMem) * 100;
    const stackPercent = (stackMem / totalMem) * 100;
    const heapPercent = (heapMem / totalMem) * 100;
    const idlePercent = (idleMem / totalMem) * 100;

    // Format memory values
    const runtimeMemFormatted = formatMemorySize(runtimeMem);
    const stackMemFormatted = formatMemorySize(stackMem);
    const heapMemFormatted = formatMemorySize(heapMem);
    const idleMemFormatted = formatMemorySize(idleMem);
    const totalMemFormatted = formatMemorySize(totalMem);

    // Define segments with all needed data
    const segments: MemorySegment[] = [
        {
            id: "other",
            label: "Runtime",
            color: "bg-yellow-600",
            mem: runtimeMemFormatted,
            percent: runtimePercent,
            desc: "Memory used by the Go runtime, including memory spans, mcache, garbage collector, and other system memory.",
        },
        {
            id: "stack",
            label: "Stack",
            color: "bg-green-600",
            mem: stackMemFormatted,
            percent: stackPercent,
            desc: "Memory used by goroutine stacks. Each goroutine has its own stack that grows and shrinks as needed.",
        },
        {
            id: "heap",
            label: "Heap",
            color: "bg-blue-600",
            mem: heapMemFormatted,
            percent: heapPercent,
            desc: "Memory currently allocated and in use by the Go heap for storing application data.",
        },
        {
            id: "idle",
            label: "Idle",
            color: "bg-gray-400",
            mem: idleMemFormatted,
            percent: idlePercent,
            desc: "Memory in the heap that is not currently in use but has been allocated from the OS. This memory can be reused by the application without requesting more from the OS.",
        },
    ];

    // Create tooltip content for each segment
    const createTooltipContent = (segment: MemorySegment) => (
        <div>
            <div className="font-medium mb-1">{segment.label}</div>
            <div className="text-secondary mb-2">
                {segment.mem.memstr} {segment.mem.memunit} ({segment.percent.toFixed(1)}% of total)
            </div>
            <div className="text-xs">
                {segment.desc}
                {segment.id === "other" && getDetailedRuntimeMemoryBreakdown(memStats)}
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
                                {segment.label}: {segment.mem.memstr} {segment.mem.memunit}
                            </span>
                        </div>
                    </RuntimeStatsTooltip>
                ))}
            </div>
            <div className="text-xs text-secondary mt-2">
                Total Process Memory: {totalMemFormatted.memstr} {totalMemFormatted.memunit}
            </div>
        </div>
    );
};
