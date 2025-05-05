// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

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
const transformDataForChart = (stats: RuntimeStatData[]) => {
    // Ensure we have at least 60 data points (1 minute at 1s intervals)
    const minDataPoints = 60;
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

    // If we have fewer than minDataPoints, pad with empty data points
    if (result.length < minDataPoints) {
        // Use the first data point as a template
        if (result.length > 0) {
            const firstPoint = result[0];
            const interval = 1000; // 1 second in milliseconds

            // Add empty data points after the actual data with null values
            // This prevents the lines from connecting to these points
            const padding = [];
            for (let i = 0; i < minDataPoints - result.length; i++) {
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

    // Skip rendering if no data
    if (!runtimeStats || runtimeStats.length === 0) {
        return (
            <div className="flex items-center justify-center h-32 text-secondary">No memory usage data available</div>
        );
    }

    // Transform data for the chart
    const chartData = transformDataForChart(runtimeStats);

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

    // Generate ticks at 30-second intervals
    const generateXAxisTicks = () => {
        if (chartData.length === 0) return [];

        const timestamps = chartData.map((d) => d.timestamp);
        const minTime = Math.min(...timestamps);
        const maxTime = Math.max(...timestamps);

        // Create ticks at 30-second intervals
        const interval = 30 * 1000; // 30 seconds in milliseconds
        const ticks = [];

        // Round down to the nearest 30-second mark
        let currentTick = Math.floor(minTime / interval) * interval;

        while (currentTick <= maxTime) {
            if (currentTick >= minTime) {
                ticks.push(currentTick);
            }
            currentTick += interval;
        }

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
                    <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
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
