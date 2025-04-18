// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Import the LegacyRuntimeStatsData type from the model
import { LegacyRuntimeStatsData } from "./runtimestats-model";

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

export interface RuntimeStatMetadata {
    statFn: (stat: LegacyRuntimeStatsData) => string | number;
    label: string;
    unit?: string;
    desc: string;
}

// Metadata for all runtime stats
export const runtimeStatsMetadata: Record<string, RuntimeStatMetadata> = {
    uptime: {
        statFn: (stat) => {
            // Calculate uptime based on current time and app start time
            // This will be replaced with actual implementation in the component
            return "Calculating...";
        },
        label: "Uptime",
        desc: "How long the application has been running since it started.",
    },
    heapMemory: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";
            return (stat.memstats.heapalloc / (1024 * 1024)).toFixed(2);
        },
        label: "Memory Usage (Heap)",
        unit: "MB",
        desc: "Current memory allocated by the heap for storing application data. This represents active memory being used by your application's data structures.",
    },
    cpuUsage: {
        statFn: (stat) => stat.cpuusage.toFixed(2),
        label: "CPU Usage",
        unit: "%",
        desc: "Percentage of CPU time being used by this Go process. High values may indicate CPU-intensive operations or potential bottlenecks.",
    },
    goroutineCount: {
        statFn: (stat) => {
            return stat.numactivegoroutines - stat.numoutriggoroutines;
        },
        label: "Goroutine Count",
        desc: "Number of goroutines currently running in the application, excluding Outrig SDK goroutines. Each goroutine is a lightweight thread managed by the Go runtime. Unexpected high counts may indicate goroutine leaks.",
    },
    currentHeapObjects: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";

            const total = stat.memstats.totalheapobj || 0;
            const free = stat.memstats.totalheapobjfree || 0;
            const current = total - free;
            return current.toLocaleString();
        },
        label: "Current Heap Objects",
        desc: "Number of live heap objects currently in memory (calculated as total allocated minus freed objects).",
    },
    totalHeapObjects: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";

            const total = stat.memstats.totalheapobj || 0;
            return total.toLocaleString();
        },
        label: "Total Heap Objects",
        desc: "Total number of heap objects allocated over the entire lifetime of the application. This counter only increases and includes objects that have been freed.",
    },
    processId: {
        statFn: (stat) => stat.pid,
        label: "Process ID",
        desc: "The operating system process identifier for this Go application.",
    },
    workingDirectory: {
        statFn: (stat) => stat.cwd,
        label: "Working Directory",
        desc: "The current working directory of the Go application process.",
    },
    goMaxProcs: {
        statFn: (stat) => stat.gomaxprocs,
        label: "GOMAXPROCS",
        desc: "Maximum number of CPUs that can be executing simultaneously. This controls the number of OS threads used for Go code execution.",
    },
    cpuCores: {
        statFn: (stat) => stat.numcpu,
        label: "CPU Cores",
        desc: "Number of CPU cores available to the Go application.",
    },
    platform: {
        statFn: (stat) => `${stat.goos}/${stat.goarch}`,
        label: "Platform",
        desc: "The operating system and architecture the Go application is running on.",
    },
    goVersion: {
        statFn: (stat) => stat.goversion,
        label: "Go Version",
        desc: "The version of Go used to build the application.",
    },
    totalMemoryAllocated: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";
            return (stat.memstats.totalalloc / (1024 * 1024)).toFixed(2);
        },
        label: "Total Memory Allocated",
        unit: "MB",
        desc: "Cumulative bytes allocated for heap objects since the process started. This counter only increases and includes memory that has been freed.",
    },
    totalProcessMemory: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";
            return (stat.memstats.sys / (1024 * 1024)).toFixed(2);
        },
        label: "Total Process Memory",
        unit: "MB",
        desc: "Total memory obtained from the OS. This includes all memory used by the Go runtime, not just the heap.",
    },
    gcCycles: {
        statFn: (stat) => {
            if (!stat.memstats) return "N/A";
            return stat.memstats.numgc;
        },
        label: "GC Cycles",
        desc: "Number of completed GC cycles since the program started. Frequent GC cycles may indicate memory pressure or allocation patterns that could be optimized.",
    },
};

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
