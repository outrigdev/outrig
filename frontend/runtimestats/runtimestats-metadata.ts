// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Helper function to format uptime in a human-readable way
export function formatUptime(milliseconds: number): string {
    if (milliseconds < 0) return "0s";
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    if (seconds < 60) {
        // Less than a minute: show seconds
        return `${seconds}s`;
    } else if (minutes < 60) {
        // Less than an hour: show minutes and seconds
        return `${minutes}m ${seconds % 60}s`;
    } else if (hours < 24) {
        // Less than a day: show hours, minutes, and seconds
        return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else {
        // More than a day: show days, hours, minutes, and seconds
        return `${days}d ${hours % 24}h ${minutes % 60}m ${seconds % 60}s`;
    }
}

// Memory chart metadata
export interface MemoryChartSegmentMetadata {
    id: string;
    label: string;
    color: string;
    valueFn: (memStats: MemoryStatsInfo) => number;
    percentFn: (memStats: MemoryStatsInfo) => number;
    desc: string;
}

export const memoryChartMetadata: MemoryChartSegmentMetadata[] = [
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
export function getDetailedOtherMemoryBreakdown(memStats: MemoryStatsInfo): string {
    return `Memory spans: ${(memStats.mspaninuse / (1024 * 1024)).toFixed(2)} MB
MCache: ${(memStats.mcacheinuse / (1024 * 1024)).toFixed(2)} MB
GC: ${(memStats.gcsys / (1024 * 1024)).toFixed(2)} MB
Other: ${(memStats.othersys / (1024 * 1024)).toFixed(2)} MB`;
}
