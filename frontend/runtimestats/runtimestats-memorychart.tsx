// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { useRef, useState, useEffect } from "react";
import { cn } from "@/util/util";
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
import { memoryChartMetadata, getDetailedOtherMemoryBreakdown } from "./runtimestats-metadata";

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

// Custom tooltip component for runtime stats
export interface RuntimeStatsTooltipProps {
    content: React.ReactNode;
    children: React.ReactNode;
    placement?: "top" | "bottom" | "left" | "right";
    className?: string;
}

export const RuntimeStatsTooltip: React.FC<RuntimeStatsTooltipProps> = ({
    children,
    content,
    placement = "top",
    className = "",
}) => {
    const [isOpen, setIsOpen] = useState(false);
    const [isVisible, setIsVisible] = useState(false);
    const timeoutRef = useRef<number | null>(null);

    const { refs, floatingStyles, context } = useFloating({
        open: isOpen,
        onOpenChange: (open) => {
            if (open) {
                // When opening, set isOpen immediately but delay visibility
                setIsOpen(true);
                // Clear any existing timeout
                if (timeoutRef.current !== null) {
                    window.clearTimeout(timeoutRef.current);
                }
                // Set a timeout to make it visible after delay
                timeoutRef.current = window.setTimeout(() => {
                    setIsVisible(true);
                }, 100); // 100ms delay before showing
            } else {
                // When closing, keep isOpen true but set visibility to false
                setIsVisible(false);
                // Clear any existing timeout
                if (timeoutRef.current !== null) {
                    window.clearTimeout(timeoutRef.current);
                }
                // Set a timeout to actually close after transition
                timeoutRef.current = window.setTimeout(() => {
                    setIsOpen(false);
                }, 100); // 100ms for fade out transition
            }
        },
        placement,
        middleware: [offset(5), flip(), shift()],
        whileElementsMounted: autoUpdate,
    });

    // Clean up timeouts on unmount
    useEffect(() => {
        return () => {
            if (timeoutRef.current !== null) {
                window.clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    const hover = useHover(context);
    const { getReferenceProps, getFloatingProps } = useInteractions([hover]);

    return (
        <>
            <div ref={refs.setReference} {...getReferenceProps()}>
                {children}
            </div>
            {isOpen && (
                <FloatingPortal>
                    <div
                        ref={refs.setFloating}
                        style={{
                            ...floatingStyles,
                            opacity: isVisible ? 1 : 0,
                            transition: "opacity 100ms ease",
                        }}
                        {...getFloatingProps()}
                        className={cn(
                            "bg-panel border border-border rounded-md px-3 py-2 text-sm text-primary shadow-md z-50 max-w-xs",
                            className
                        )}
                    >
                        {content}
                    </div>
                </FloatingPortal>
            )}
        </>
    );
};