// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { TimestampDot } from "@/elements/timestampdot";
import { useOutrigModel } from "@/util/hooks";
import { formatMemorySize, formatTimeOffset } from "@/util/util";
import { useAtomValue } from "jotai";
import React from "react";
import { MemoryAreaChart } from "./runtimestats-memoryareachart";
import { MemoryUsageChart } from "./runtimestats-memorychart";
import { CombinedStatsData, RuntimeStatsModel } from "./runtimestats-model";
import { RuntimeStatsTooltip } from "./tooltip";

// Base component for stat items to ensure consistent styling
interface BaseStatItemProps {
    label: string;
    description: string;
    children: React.ReactNode;
    className?: string;
}

const BaseStatItem: React.FC<BaseStatItemProps> = ({ label, description, children, className = "" }) => {
    const tooltipContent = (
        <div>
            <div className="font-medium mb-1">{label}</div>
            <div className="text-xs">{description}</div>
        </div>
    );

    return (
        <RuntimeStatsTooltip content={tooltipContent}>
            <div className="p-4 border border-border rounded-md bg-panel">
                <div className="text-sm text-secondary mb-1">{label}</div>
                <div className={`text-2xl font-semibold text-primary ${className}`}>{children}</div>
            </div>
        </RuntimeStatsTooltip>
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
    const uptimeText = formatTimeOffset(endTime, startTime);

    const isRunning = appRunInfo.isrunning && appRunInfo.status === "running";

    return (
        <BaseStatItem
            label="Uptime"
            description="How long the application has been running since it started."
            className="flex items-center"
        >
            {uptimeText}
            {isRunning && <span className="ml-2 inline-block w-2 h-2 rounded-full bg-green-500" title="Running" />}
        </BaseStatItem>
    );
};

const StatItem: React.FC<StatItemProps> = ({ value, label, unit, desc }) => {
    return (
        <BaseStatItem label={label} description={desc}>
            {value}
            {unit && <span className="text-sm text-secondary ml-1">{unit}</span>}
        </BaseStatItem>
    );
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

// Section header component
interface SectionHeaderProps {
    title: string;
}

const SectionHeader: React.FC<SectionHeaderProps> = ({ title }) => {
    return (
        <div className="col-span-full mt-1">
            <h3 className="text-xs font-medium text-secondary uppercase tracking-wider">{title}</h3>
        </div>
    );
};

// MetricsGrid component that displays all the stat items
interface MetricsGridProps {
    stats: CombinedStatsData;
    appRunInfo: AppRunInfo;
}

const MetricsGrid: React.FC<MetricsGridProps> = ({ stats, appRunInfo }) => {
    // Format memory values once and reuse
    const heapMemory = formatMemorySize(stats.memstats.heapalloc);
    const totalMemory = formatMemorySize(stats.memstats.totalalloc);
    const sysMemory = formatMemorySize(stats.memstats.sys);

    return (
        <div className="grid grid-cols-3 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {/* Current section */}
            <SectionHeader title="Current" />

            {/* First row: Uptime, Goroutine Count, CPU Usage */}
            <UptimeStatItem appRunInfo={appRunInfo} />

            <StatItem
                value={stats.numactivegoroutines - stats.numoutriggoroutines}
                label="Goroutine Count"
                desc="Number of goroutines currently running in the application, excluding Outrig SDK goroutines. Each goroutine is a lightweight thread managed by the Go runtime. Unexpected high counts may indicate goroutine leaks."
            />

            {/* Second row: Total Process Memory, Heap Memory, Heap Objects */}
            <StatItem
                value={sysMemory.memstr}
                label="Total Process Memory"
                unit={sysMemory.memunit}
                desc="Total memory obtained from the OS. This includes all memory used by the Go runtime, not just the heap."
            />

            <StatItem
                value={heapMemory.memstr}
                label="Heap Memory"
                unit={heapMemory.memunit}
                desc="Current memory allocated by the heap for storing application data. This represents active memory being used by your application's data structures."
            />

            <StatItem
                value={(stats.memstats.totalheapobj - (stats.memstats.totalheapobjfree || 0)).toLocaleString()}
                label="Heap Objects"
                desc="Number of live heap objects currently in memory (calculated as total allocated minus freed objects)."
            />

            {/* Lifetime section */}
            <SectionHeader title="Lifetime" />

            <StatItem
                value={stats.memstats.totalheapobj.toLocaleString()}
                label="Lifetime Heap Objects"
                desc="Total number of heap objects allocated over the entire lifetime of the application. This counter only increases and includes objects that have been freed."
            />

            <StatItem
                value={totalMemory.memstr}
                label="Lifetime Allocation"
                unit={totalMemory.memunit}
                desc="Cumulative bytes allocated for heap objects since the process started. This counter only increases and includes memory that has been freed."
            />

            <StatItem
                value={stats.memstats.numgc.toLocaleString()}
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
                <div className="text-sm text-secondary font-medium mb-2">Memory Breakdown</div>
                <MemoryAreaChart model={model} height={300} />
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
