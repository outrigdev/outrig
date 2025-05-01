// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { TimestampDot } from "@/elements/timestampdot";
import { useOutrigModel } from "@/util/hooks";
import { useAtomValue } from "jotai";
import React from "react";
import { MemoryUsageChart, RuntimeStatsTooltip } from "./runtimestats-memorychart";
import { formatUptime } from "./runtimestats-metadata";
import { CombinedStatsData, RuntimeStatsModel } from "./runtimestats-model";

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
    stats: CombinedStatsData;
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
