// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { formatMemorySize } from "@/util/util";
import { useAtomValue } from "jotai";
import React from "react";
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { RuntimeStatsModel } from "./runtimestats-model";

// Memory area chart component
interface MemoryAreaChartProps {
    model: RuntimeStatsModel;
    height?: number;
}

// Custom tooltip component for the area chart
const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
        const timestamp = new Date(label).toLocaleTimeString();
        const totalMem = formatMemorySize(payload[0].value + payload[1].value + payload[2].value + payload[3].value);

        return (
            <div className="bg-panel border border-border rounded-md px-3 py-2 text-sm text-primary shadow-md">
                <div className="font-medium mb-1">{timestamp}</div>
                <div className="mb-2">
                    Total: {totalMem.memstr} {totalMem.memunit}
                </div>
                {payload.map((entry: any, index: number) => {
                    const mem = formatMemorySize(entry.value);
                    return (
                        <div key={`item-${index}`} className="flex items-center mb-1">
                            <div className="w-3 h-3 mr-1 rounded-sm" style={{ backgroundColor: entry.fill }}></div>
                            <span>
                                {entry.name}: {mem.memstr} {mem.memunit}
                            </span>
                        </div>
                    );
                })}
            </div>
        );
    }

    return null;
};

// Helper function to transform runtime stats data for the chart
const transformDataForChart = (stats: RuntimeStatData[], isRunning: boolean) => {
    let result = stats.map((stat) => {
        const runtimeMem =
            stat.memstats.mspaninuse + stat.memstats.mcacheinuse + stat.memstats.gcsys + stat.memstats.othersys;
        const stackMem = stat.memstats.stackinuse;
        const heapMem = stat.memstats.heapinuse;
        const idleMem = stat.memstats.heapidle;

        return {
            timestamp: stat.ts,
            Runtime: runtimeMem,
            Stack: stackMem,
            Heap: heapMem,
            Idle: idleMem,
        };
    });

    // Only pad with empty data points if the program is still running
    if (isRunning && result.length > 0) {
        // Determine target size based on current data length using math
        // This creates fixed size increments (15, 30, 60, 90, 120, etc.)
        let targetSize = 15; // Start with minimum of 15 points
        
        if (result.length > 15) {
            if (result.length <= 30) {
                targetSize = 30;
            } else {
                // For values > 30, we use multiples of 30
                // Calculate how many complete 30s we need
                const thirtyMultiple = Math.ceil(result.length / 30);
                targetSize = thirtyMultiple * 30;
                
                // Cap at 600 (10 minutes at 1s intervals)
                targetSize = Math.min(targetSize, 600);
            }
        }
        
        // Only pad if we need to
        if (result.length < targetSize) {
            const firstPoint = result[0];
            const interval = 1000; // 1 second in milliseconds

            // Add empty data points after the actual data with null values
            // This prevents the lines from connecting to these points
            const padding = [];
            for (let i = 0; i < targetSize - result.length; i++) {
                padding.push({
                    timestamp: firstPoint.timestamp + result.length * interval + i * interval,
                    Runtime: null,
                    Stack: null,
                    Heap: null,
                    Idle: null,
                });
            }

            result = [...result, ...padding];
        }
    }

    return result;
};

export const MemoryAreaChart: React.FC<MemoryAreaChartProps> = ({ model, height = 300 }) => {
    // Get runtime stats from the model's atom
    const runtimeStats = useAtomValue(model.allRuntimeStats);

    // Get app run info to check if the program is running
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(model.appRunId);
    const appRunInfo = useAtomValue(appRunInfoAtom);

    // Skip rendering if no data
    if (!runtimeStats || runtimeStats.length === 0) {
        return null;
    }

    // Check if we should hide the chart:
    // 1. Program is not running, AND
    // 2. There is less than 5s of data
    if (appRunInfo && appRunInfo.status !== "running" && runtimeStats.length > 0) {
        // Calculate time span of the data
        const timestamps = runtimeStats.map((stat) => stat.ts);
        const minTime = Math.min(...timestamps);
        const maxTime = Math.max(...timestamps);
        const timeSpanMs = maxTime - minTime;

        // Hide chart if time span is less than 5 seconds (5000ms)
        if (timeSpanMs < 5000) {
            return null;
        }
    }

    // Transform data for the chart
    const isRunning = appRunInfo && appRunInfo.status === "running";
    const chartData = transformDataForChart(runtimeStats, isRunning);

    // Define colors for each area (matching the existing chart colors)
    const areaColors = {
        Runtime: "#ca8a04", // yellow-600
        Stack: "#16a34a", // green-600
        Heap: "#2563eb", // blue-600
        Idle: "#9ca3af50", // gray-400 with 50% transparency
    };

    // Calculate the maximum memory value for the Y-axis
    const maxMemory = Math.max(...chartData.map((data) => data.Runtime + data.Stack + data.Heap + data.Idle));

    // Format the Y-axis tick values
    const formatYAxis = (value: number) => {
        const mem = formatMemorySize(value);
        return `${mem.memstr}${mem.memunit}`;
    };

    // Format the X-axis tick values using locale-aware time formatting
    const formatXAxis = (timestamp: number) => {
        const date = new Date(timestamp);
        // Use locale-aware time formatting with consistent precision
        // Include seconds since we have 30-second ticks
        const timeString = date.toLocaleTimeString(undefined, {
            hour: "numeric",
            minute: "2-digit",
            second: "2-digit",
        });

        // Convert AM/PM to lowercase and remove space before it
        return timeString.replace(/\s*(AM|PM)\b/g, (match) => match.toLowerCase().trim());
    };

    // Generate ticks at 30-second intervals, always including the start and end time
    const generateXAxisTicks = () => {
        if (chartData.length === 0) return [];

        const timestamps = chartData.map((d) => d.timestamp);
        const minTime = Math.min(...timestamps);
        const maxTime = Math.max(...timestamps);

        // Create ticks at 30-second intervals
        const interval = 30 * 1000; // 30 seconds in milliseconds
        const ticks = [minTime]; // Always include the start time

        // Round down to the nearest 30-second mark
        let currentTick = Math.floor(minTime / interval) * interval;
        
        // If the rounded tick is the same as minTime, skip to the next interval
        if (currentTick === minTime) {
            currentTick += interval;
        } else if (currentTick < minTime) {
            currentTick += interval;
        }

        while (currentTick < maxTime) {
            ticks.push(currentTick);
            currentTick += interval;
        }
        
        // Always include the end time
        ticks.push(maxTime);

        return ticks;
    };

    // Create tooltip content for the legend
    const createTooltipContent = (label: string) => {
        let description = "";
        switch (label) {
            case "Runtime":
                description =
                    "Memory used by the Go runtime, including memory spans, mcache, garbage collector, and other system memory.";
                break;
            case "Stack":
                description =
                    "Memory used by goroutine stacks. Each goroutine has its own stack that grows and shrinks as needed.";
                break;
            case "Heap":
                description = "Memory currently allocated and in use by the Go heap for storing application data.";
                break;
            case "Idle":
                description =
                    "Memory in the heap that is not currently in use but has been allocated from the OS. This memory can be reused by the application without requesting more from the OS.";
                break;
        }

        return (
            <div>
                <div className="font-medium mb-1">{label}</div>
                <div className="text-xs">{description}</div>
            </div>
        );
    };

    return (
        <div className="w-full" style={{ height }}>
            <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                    <CartesianGrid strokeWidth={1} stroke="#444" />
                    <XAxis
                        dataKey="timestamp"
                        type="number"
                        scale="time"
                        domain={["dataMin", "dataMax"]}
                        tickFormatter={formatXAxis}
                        ticks={generateXAxisTicks()}
                        tick={{ fontSize: 10, fill: "#9ca3af" }}
                        height={30}
                        tickMargin={8}
                        axisLine={{ stroke: "#666" }}
                        tickLine={{ stroke: "#666" }}
                        minTickGap={30}
                    />
                    <YAxis
                        tickFormatter={formatYAxis}
                        tick={{ fontSize: 10, fill: "#9ca3af" }}
                        width={60}
                        axisLine={{ stroke: "#666" }}
                        tickLine={{ stroke: "#666" }}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    {/* Order matters for stacking - first one is at the bottom */}
                    <Area
                        type="monotone"
                        dataKey="Runtime"
                        stackId="1"
                        stroke="transparent"
                        fill={areaColors.Runtime}
                        isAnimationActive={false}
                    />
                    <Area
                        type="monotone"
                        dataKey="Stack"
                        stackId="1"
                        stroke="transparent"
                        fill={areaColors.Stack}
                        isAnimationActive={false}
                    />
                    <Area
                        type="monotone"
                        dataKey="Heap"
                        stackId="1"
                        stroke="transparent"
                        fill={areaColors.Heap}
                        isAnimationActive={false}
                    />
                    <Area
                        type="monotone"
                        dataKey="Idle"
                        stackId="1"
                        stroke="transparent"
                        fill={areaColors.Idle}
                        isAnimationActive={false}
                    />
                </AreaChart>
            </ResponsiveContainer>
        </div>
    );
};
