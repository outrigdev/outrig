import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { useOutrigModel } from "@/util/hooks";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import React, { useState, useRef, useEffect } from "react";
import { RuntimeStatsModel } from "./runtimestats-model";
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
    className = "" 
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
            <div ref={refs.setReference} {...getReferenceProps()} className="cursor-pointer">
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
    // Calculate percentages for the chart
    const heapInUsePercent = (memStats.heapinuse / memStats.sys) * 100;
    const stackInUsePercent = (memStats.stackinuse / memStats.sys) * 100;
    const otherInUsePercent = ((memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / memStats.sys) * 100;
    const idlePercent = (memStats.heapidle / memStats.sys) * 100;

    // Format memory values
    const heapInUseMB = (memStats.heapinuse / (1024 * 1024)).toFixed(2);
    const stackInUseMB = (memStats.stackinuse / (1024 * 1024)).toFixed(2);
    const otherMemoryMB = ((memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / (1024 * 1024)).toFixed(2);
    const heapIdleMB = (memStats.heapidle / (1024 * 1024)).toFixed(2);

    // Create detailed tooltip content for each memory type
    const heapTooltipContent = (
        <div>
            <div className="font-medium mb-1">Heap Memory In Use</div>
            <div className="text-secondary mb-2">{heapInUseMB} MB ({heapInUsePercent.toFixed(1)}% of total)</div>
            <div className="text-xs">Memory currently allocated and in use by the Go heap for storing application data.</div>
        </div>
    );

    const stackTooltipContent = (
        <div>
            <div className="font-medium mb-1">Stack Memory</div>
            <div className="text-secondary mb-2">{stackInUseMB} MB ({stackInUsePercent.toFixed(1)}% of total)</div>
            <div className="text-xs">Memory used by goroutine stacks. Each goroutine has its own stack that grows and shrinks as needed.</div>
        </div>
    );

    const otherTooltipContent = (
        <div>
            <div className="font-medium mb-1">Other Runtime Memory</div>
            <div className="text-secondary mb-2">{otherMemoryMB} MB ({otherInUsePercent.toFixed(1)}% of total)</div>
            <div className="text-xs">
                <div>Memory spans: {(memStats.mspaninuse / (1024 * 1024)).toFixed(2)} MB</div>
                <div>MCache: {(memStats.mcacheinuse / (1024 * 1024)).toFixed(2)} MB</div>
                <div>GC: {(memStats.gcsys / (1024 * 1024)).toFixed(2)} MB</div>
                <div>Other: {(memStats.othersys / (1024 * 1024)).toFixed(2)} MB</div>
            </div>
        </div>
    );

    const idleTooltipContent = (
        <div>
            <div className="font-medium mb-1">Heap Idle Memory</div>
            <div className="text-secondary mb-2">{heapIdleMB} MB ({idlePercent.toFixed(1)}% of total)</div>
            <div className="text-xs">Memory in the heap that is not currently in use but has been allocated from the OS. This memory can be reused by the application without requesting more from the OS.</div>
        </div>
    );

    // State to track which segment is being hovered
    const [hoveredSegment, setHoveredSegment] = useState<string | null>(null);
    const [tooltipOpen, setTooltipOpen] = useState(false);
    const chartRef = useRef<HTMLDivElement>(null);
    const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 });
    const [tooltipContent, setTooltipContent] = useState<React.ReactNode>(null);

    // Handle mouse movement over the chart
    const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
        if (!chartRef.current) return;
        
        const rect = chartRef.current.getBoundingClientRect();
        const x = e.clientX - rect.left; // x position within the element
        const totalWidth = rect.width;
        const relativePosition = x / totalWidth;
        
        // Calculate which segment the mouse is over based on cumulative widths
        const heapInUseEnd = heapInUsePercent / 100;
        const stackInUseEnd = heapInUseEnd + (stackInUsePercent / 100);
        const otherInUseEnd = stackInUseEnd + (otherInUsePercent / 100);
        
        let newHoveredSegment: string | null = null;
        let content: React.ReactNode = null;
        
        if (relativePosition <= heapInUseEnd) {
            newHoveredSegment = 'heap';
            content = heapTooltipContent;
        } else if (relativePosition <= stackInUseEnd) {
            newHoveredSegment = 'stack';
            content = stackTooltipContent;
        } else if (relativePosition <= otherInUseEnd) {
            newHoveredSegment = 'other';
            content = otherTooltipContent;
        } else {
            newHoveredSegment = 'idle';
            content = idleTooltipContent;
        }
        
        if (newHoveredSegment !== hoveredSegment) {
            setHoveredSegment(newHoveredSegment);
            setTooltipContent(content);
        }
        
        // Position tooltip near the cursor
        setTooltipPosition({ 
            x: e.clientX, 
            y: e.clientY - 10 // Offset slightly above cursor
        });
    };

    return (
        <div>
            <div 
                className="relative flex h-6 w-full rounded-md overflow-hidden mb-2"
                ref={chartRef}
                onMouseMove={handleMouseMove}
                onMouseEnter={() => setTooltipOpen(true)}
                onMouseLeave={() => {
                    setTooltipOpen(false);
                    setHoveredSegment(null);
                }}
            >
                <div 
                    className="bg-blue-600 h-full cursor-pointer" 
                    style={{ width: `${heapInUsePercent}%` }} 
                />
                <div 
                    className="bg-green-600 h-full cursor-pointer" 
                    style={{ width: `${stackInUsePercent}%` }} 
                />
                <div 
                    className="bg-yellow-600 h-full cursor-pointer" 
                    style={{ width: `${otherInUsePercent}%` }} 
                />
                <div 
                    className="bg-gray-400 h-full cursor-pointer" 
                    style={{ width: `${idlePercent}%` }} 
                />
                
                {tooltipOpen && hoveredSegment && (
                    <FloatingPortal>
                        <div 
                            className="fixed z-50 bg-panel border border-border rounded-md px-3 py-2 text-sm text-primary shadow-md max-w-xs"
                            style={{
                                left: `${tooltipPosition.x}px`,
                                top: `${tooltipPosition.y - 100}px`, // Position well above the cursor
                                transform: 'translateX(-50%)',
                                opacity: 1,
                                pointerEvents: 'none', // Prevent the tooltip from interfering with mouse events
                                width: '250px',
                            }}
                        >
                            {tooltipContent}
                        </div>
                    </FloatingPortal>
                )}
            </div>
            <div className="flex flex-wrap text-xs gap-3 mb-2">
                <RuntimeStatsTooltip content={heapTooltipContent}>
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-blue-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Heap In Use: {heapInUseMB} MB</span>
                    </div>
                </RuntimeStatsTooltip>
                <RuntimeStatsTooltip content={stackTooltipContent}>
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-green-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Stack: {stackInUseMB} MB</span>
                    </div>
                </RuntimeStatsTooltip>
                <RuntimeStatsTooltip content={otherTooltipContent}>
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-yellow-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Other: {otherMemoryMB} MB</span>
                    </div>
                </RuntimeStatsTooltip>
                <RuntimeStatsTooltip content={idleTooltipContent}>
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-gray-400 mr-1 rounded-sm"></div>
                        <span className="text-primary">Idle: {heapIdleMB} MB</span>
                    </div>
                </RuntimeStatsTooltip>
            </div>
            <div className="text-xs text-secondary mt-2">
                Total Process Memory: {(memStats.sys / (1024 * 1024)).toFixed(2)} MB
            </div>
        </div>
    );
};

// Component for displaying a single stat
interface StatItemProps {
    label: string;
    value: string | number;
    unit?: string;
    tooltip?: React.ReactNode;
}

const StatItem: React.FC<StatItemProps> = ({ label, value, unit, tooltip }) => {
    const content = (
        <div className="mb-4 p-4 border border-border rounded-md bg-panel">
            <div className="text-sm text-secondary mb-1">{label}</div>
            <div className="text-2xl font-semibold text-primary">
                {value}
                {unit && <span className="text-sm text-secondary ml-1">{unit}</span>}
            </div>
        </div>
    );

    if (tooltip) {
        return (
            <RuntimeStatsTooltip content={tooltip}>
                {content}
            </RuntimeStatsTooltip>
        );
    }

    return content;
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

// Content component that displays the runtime stats
interface RuntimeStatsContentProps {
    model: RuntimeStatsModel;
}

const RuntimeStatsContent: React.FC<RuntimeStatsContentProps> = ({ model }) => {
    const stats = useAtomValue(model.runtimeStats);
    const isRefreshing = useAtomValue(model.isRefreshing);

    if (isRefreshing && !stats) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="flex items-center gap-2 text-primary">
                    <span>Loading runtime stats...</span>
                </div>
            </div>
        );
    }

    if (!stats) {
        return <div className="flex items-center justify-center h-full text-secondary">No runtime stats available</div>;
    }

    // Format the timestamp
    const formattedTime = new Date(stats.ts).toLocaleTimeString();

    // Calculate memory usage in MB for display
    const heapAllocMB = stats.memstats ? (stats.memstats.heapalloc / (1024 * 1024)).toFixed(2) : "0";
    const totalAllocMB = stats.memstats ? (stats.memstats.totalalloc / (1024 * 1024)).toFixed(2) : "0";
    const sysMB = stats.memstats ? (stats.memstats.sys / (1024 * 1024)).toFixed(2) : "0";

    return (
        <div className="w-full h-full overflow-auto p-4">
            <div className="text-sm text-secondary mb-4">Last updated: {formattedTime}</div>

            {/* Memory usage visualization */}
            {stats.memstats && (
                <div className="mb-6 p-4 border border-border rounded-md bg-panel">
                    <div className="text-sm text-secondary font-medium mb-2">Memory Usage Breakdown</div>
                    <MemoryUsageChart memStats={stats.memstats} />
                </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <StatItem 
                    label="Memory Usage (Heap)" 
                    value={heapAllocMB} 
                    unit="MB" 
                    tooltip={
                        <div>
                            <div className="font-medium mb-1">Heap Memory Usage</div>
                            <div className="text-xs">
                                <p className="mb-1">Current memory allocated by the heap for storing application data.</p>
                                <p>This represents active memory being used by your application's data structures.</p>
                            </div>
                        </div>
                    }
                />
                <StatItem 
                    label="CPU Usage" 
                    value={stats.cpuusage.toFixed(2)} 
                    unit="%" 
                    tooltip={
                        <div>
                            <div className="font-medium mb-1">CPU Usage</div>
                            <div className="text-xs">
                                <p className="mb-1">Percentage of CPU time being used by this Go process.</p>
                                <p>High values may indicate CPU-intensive operations or potential bottlenecks.</p>
                            </div>
                        </div>
                    }
                />
                <StatItem 
                    label="Goroutine Count" 
                    value={stats.goroutinecount}
                    tooltip={
                        <div>
                            <div className="font-medium mb-1">Active Goroutines</div>
                            <div className="text-xs">
                                <p className="mb-1">Number of goroutines currently running in the application.</p>
                                <p>Each goroutine is a lightweight thread managed by the Go runtime.</p>
                                <p>Unexpected high counts may indicate goroutine leaks.</p>
                            </div>
                        </div>
                    }
                />
                <StatItem label="Process ID" value={stats.pid} />
                <StatItem label="Working Directory" value={stats.cwd} />
                <StatItem 
                    label="GOMAXPROCS" 
                    value={stats.gomaxprocs}
                    tooltip={
                        <div>
                            <div className="font-medium mb-1">GOMAXPROCS</div>
                            <div className="text-xs">
                                <p className="mb-1">Maximum number of CPUs that can be executing simultaneously.</p>
                                <p>This controls the number of OS threads used for Go code execution.</p>
                            </div>
                        </div>
                    }
                />
                <StatItem label="CPU Cores" value={stats.numcpu} />
                <StatItem label="Platform" value={`${stats.goos}/${stats.goarch}`} />
                <StatItem label="Go Version" value={stats.goversion} />
                
                {stats.memstats && (
                    <>
                        <StatItem 
                            label="Total Memory Allocated" 
                            value={totalAllocMB} 
                            unit="MB" 
                            tooltip={
                                <div>
                                    <div className="font-medium mb-1">Total Memory Allocated</div>
                                    <div className="text-xs">
                                        <p className="mb-1">Cumulative bytes allocated for heap objects since the process started.</p>
                                        <p>This counter only increases and includes memory that has been freed.</p>
                                    </div>
                                </div>
                            }
                        />
                        <StatItem 
                            label="Total Process Memory" 
                            value={sysMB} 
                            unit="MB" 
                            tooltip={
                                <div>
                                    <div className="font-medium mb-1">Total Process Memory</div>
                                    <div className="text-xs">
                                        <p className="mb-1">Total memory obtained from the OS.</p>
                                        <p>This includes all memory used by the Go runtime, not just the heap.</p>
                                    </div>
                                </div>
                            }
                        />
                        <StatItem 
                            label="GC Cycles" 
                            value={stats.memstats.numgc}
                            tooltip={
                                <div>
                                    <div className="font-medium mb-1">Garbage Collection Cycles</div>
                                    <div className="text-xs">
                                        <p className="mb-1">Number of completed GC cycles since the program started.</p>
                                        <p>Frequent GC cycles may indicate memory pressure or allocation patterns that could be optimized.</p>
                                    </div>
                                </div>
                            }
                        />
                    </>
                )}
            </div>
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
