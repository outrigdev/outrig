// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { TimestampDot } from "@/elements/timestampdot";
import { useOutrigModel } from "@/util/hooks";
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
import { useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { formatUptime, getDetailedOtherMemoryBreakdown, memoryChartMetadata } from "./runtimestats-metadata";
import { LegacyRuntimeStatsData, RuntimeStatsModel } from "./runtimestats-model";

// Custom tooltip component for runtime stats
interface RuntimeStatsTooltipProps {
    content: React.ReactNode;
    children: React.ReactNode;
    placement?: "top" | "bottom" | "left" | "right";
    className?: string;
}

const RuntimeStatsTooltip: React.FC<RuntimeStatsTooltipProps> = ({
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

// Memory usage chart component
interface MemoryUsageChartProps {
    memStats: MemoryStatsInfo;
}

const MemoryUsageChart: React.FC<MemoryUsageChartProps> = ({ memStats }) => {
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

// Component for displaying a single stat
interface StatItemProps {
    value: string | number;
    label: string;
    unit?: string;
    desc: string;
}

// Component for displaying uptime
interface UptimeStatItemProps {
    appRunInfo: AppRunInfo;
}

const UptimeStatItem: React.FC<UptimeStatItemProps> = ({ appRunInfo }) => {
    // Calculate uptime
    const startTime = appRunInfo.starttime;
    const endTime = appRunInfo.isrunning && appRunInfo.status === "running" ? Date.now() : appRunInfo.lastmodtime;
    const uptimeDuration = endTime - startTime;
    const uptimeText = formatUptime(uptimeDuration);

    const isRunning = appRunInfo.isrunning && appRunInfo.status === "running";

    const tooltipContent = (
        <div>
            <div className="font-medium mb-1">Uptime</div>
            <div className="text-xs">How long the application has been running since it started.</div>
        </div>
    );

    const content = (
        <div className="mb-4 p-4 border border-border rounded-md bg-panel">
            <div className="text-sm text-secondary mb-1">Uptime</div>
            <div className="text-2xl font-semibold text-primary flex items-center">
                {uptimeText}
                {isRunning && <span className="ml-2 inline-block w-2 h-2 rounded-full bg-green-500" title="Running" />}
            </div>
        </div>
    );

    return <RuntimeStatsTooltip content={tooltipContent}>{content}</RuntimeStatsTooltip>;
};

const StatItem: React.FC<StatItemProps> = ({ value, label, unit, desc }) => {
    const tooltipContent = (
        <div>
            <div className="font-medium mb-1">{label}</div>
            <div className="text-xs">{desc}</div>
        </div>
    );

    const content = (
        <div className="mb-4 p-4 border border-border rounded-md bg-panel">
            <div className="text-sm text-secondary mb-1">{label}</div>
            <div className="text-2xl font-semibold text-primary">
                {value}
                {unit && <span className="text-sm text-secondary ml-1">{unit}</span>}
            </div>
        </div>
    );

    return <RuntimeStatsTooltip content={tooltipContent}>{content}</RuntimeStatsTooltip>;
};

// Header component with refresh button
interface RuntimeStatsHeaderProps {
    model: RuntimeStatsModel;
}

const RuntimeStatsHeader: React.FC<RuntimeStatsHeaderProps> = ({ model }) => {
    return (
        <div className="py-1 px-4 border-b border-border flex items-center justify-between">
            <h2 className="text-primary text-lg">Runtime Stats</h2>
            <div className="flex items-center">
                <AutoRefreshButton autoRefreshAtom={model.autoRefresh} onToggle={() => model.toggleAutoRefresh()} />
                <RefreshButton
                    isRefreshingAtom={model.isRefreshing}
                    onRefresh={() => model.refresh()}
                    tooltipContent="Refresh runtime stats"
                    size={16}
                />
            </div>
        </div>
    );
};

// MetricsGrid component that displays all the stat items
interface MetricsGridProps {
    stats: LegacyRuntimeStatsData;
    appRunInfo: AppRunInfo;
}

const MetricsGrid: React.FC<MetricsGridProps> = ({ stats, appRunInfo }) => {
    return (
        <div className="grid grid-cols-3 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {/* Uptime stat with status indicator */}
            <UptimeStatItem appRunInfo={appRunInfo} />

            {/* Manually create StatItem components for each metric */}
            <StatItem
                value={(stats.memstats.heapalloc / (1024 * 1024)).toFixed(2)}
                label="Memory Usage (Heap)"
                unit="MB"
                desc="Current memory allocated by the heap for storing application data. This represents active memory being used by your application's data structures."
            />

            <StatItem
                value={stats.cpuusage.toFixed(2)}
                label="CPU Usage"
                unit="%"
                desc="Percentage of CPU time being used by this Go process. High values may indicate CPU-intensive operations or potential bottlenecks."
            />

            <StatItem
                value={stats.numactivegoroutines - stats.numoutriggoroutines}
                label="Goroutine Count"
                desc="Number of goroutines currently running in the application, excluding Outrig SDK goroutines. Each goroutine is a lightweight thread managed by the Go runtime. Unexpected high counts may indicate goroutine leaks."
            />

            <StatItem
                value={(stats.memstats.totalheapobj - (stats.memstats.totalheapobjfree || 0)).toLocaleString()}
                label="Current Heap Objects"
                desc="Number of live heap objects currently in memory (calculated as total allocated minus freed objects)."
            />

            <StatItem
                value={stats.memstats.totalheapobj.toLocaleString()}
                label="Total Heap Objects"
                desc="Total number of heap objects allocated over the entire lifetime of the application. This counter only increases and includes objects that have been freed."
            />

            <StatItem
                value={(stats.memstats.totalalloc / (1024 * 1024)).toFixed(2)}
                label="Total Memory Allocated"
                unit="MB"
                desc="Cumulative bytes allocated for heap objects since the process started. This counter only increases and includes memory that has been freed."
            />

            <StatItem
                value={(stats.memstats.sys / (1024 * 1024)).toFixed(2)}
                label="Total Process Memory"
                unit="MB"
                desc="Total memory obtained from the OS. This includes all memory used by the Go runtime, not just the heap."
            />

            <StatItem
                value={stats.memstats.numgc}
                label="GC Cycles"
                desc="Number of completed GC cycles since the program started. Frequent GC cycles may indicate memory pressure or allocation patterns that could be optimized."
            />
        </div>
    );
};

// Content component that displays the runtime stats
interface RuntimeStatsContentProps {
    model: RuntimeStatsModel;
}

const RuntimeStatsContent: React.FC<RuntimeStatsContentProps> = ({ model }) => {
    const stats = useAtomValue(model.runtimeStats);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const appRunInfoAtom = React.useMemo(() => AppModel.getAppRunInfoAtom(model.appRunId), [model.appRunId]);
    const appRunInfo = useAtomValue(appRunInfoAtom);

    if (isRefreshing && !stats) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="flex items-center gap-2 text-primary">
                    <span>Loading runtime stats...</span>
                </div>
            </div>
        );
    }

    if (!stats || !stats.memstats || !appRunInfo) {
        return <div className="flex items-center justify-center h-full text-secondary">No runtime stats available</div>;
    }

    const formattedTime = new Date(stats.ts).toLocaleTimeString();

    // Information items that should be displayed in a more informational way
    const infoItems = [
        { key: "processId", label: "Process ID", value: stats.pid },
        { key: "workingDirectory", label: "Working Directory", value: stats.cwd },
        { key: "goMaxProcs", label: "GOMAXPROCS", value: stats.gomaxprocs },
        { key: "cpuCores", label: "CPU Cores", value: stats.numcpu },
        { key: "platform", label: "Platform", value: `${stats.goos}/${stats.goarch}` },
        { key: "goVersion", label: "Go Version", value: stats.goversion },
        { key: "moduleName", label: "Module", value: appRunInfo.modulename },
        { key: "executable", label: "Executable", value: appRunInfo.executable },
        { key: "sdkVersion", label: "Outrig SDK Version", value: appRunInfo.outrigsdkversion },
    ];

    return (
        <div className="w-full h-full overflow-auto p-4">
            <div className="flex items-center gap-2 text-sm text-secondary mb-4">
                <TimestampDot timestamp={stats.ts} />
                <span>Last updated: {formattedTime}</span>
            </div>

            {/* Memory usage visualization */}
            <div className="mb-6 p-4 border border-border rounded-md bg-panel">
                <div className="text-sm text-secondary font-medium mb-2">Memory Usage Breakdown</div>
                <MemoryUsageChart memStats={stats.memstats} />
            </div>

            {/* Information panel */}
            <div className="mb-6 p-4 border border-border rounded-md bg-panel">
                <div className="text-sm text-secondary font-medium mb-3">Application Information</div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-3">
                    {infoItems.map((item) => (
                        <div key={item.key} className="flex flex-col">
                            <div className="text-xs text-secondary">{item.label}</div>
                            <div className="text-sm text-primary font-mono truncate" title={String(item.value)}>
                                {item.value}
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Metrics grid */}
            <MetricsGrid stats={stats} appRunInfo={appRunInfo} />
        </div>
    );
};

// Main runtime stats component that composes the sub-components
interface RuntimeStatsProps {
    appRunId: string;
}

export const RuntimeStats: React.FC<RuntimeStatsProps> = ({ appRunId }) => {
    const model = useOutrigModel(RuntimeStatsModel, appRunId);

    if (!model) {
        return null;
    }

    return (
        <div className="w-full h-full flex flex-col">
            <RuntimeStatsHeader model={model} />
            <RuntimeStatsContent model={model} />
        </div>
    );
};
