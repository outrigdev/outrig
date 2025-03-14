import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { useOutrigModel } from "@/util/hooks";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import React from "react";
import { RuntimeStatsModel } from "./runtimestats-model";

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

    return (
        <div>
            <div className="flex h-6 w-full rounded-md overflow-hidden mb-2">
                <div 
                    className="bg-blue-600 h-full" 
                    style={{ width: `${heapInUsePercent}%` }} 
                    title={`Heap In Use: ${(memStats.heapinuse / (1024 * 1024)).toFixed(2)} MB`}
                />
                <div 
                    className="bg-green-600 h-full" 
                    style={{ width: `${stackInUsePercent}%` }} 
                    title={`Stack In Use: ${(memStats.stackinuse / (1024 * 1024)).toFixed(2)} MB`}
                />
                <div 
                    className="bg-yellow-600 h-full" 
                    style={{ width: `${otherInUsePercent}%` }} 
                    title={`Other Memory: ${((memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / (1024 * 1024)).toFixed(2)} MB`}
                />
                <div 
                    className="bg-gray-400 h-full" 
                    style={{ width: `${idlePercent}%` }} 
                    title={`Heap Idle: ${(memStats.heapidle / (1024 * 1024)).toFixed(2)} MB`}
                />
            </div>
            <div className="flex flex-wrap text-xs gap-3 mb-2">
                <Tooltip content="Memory currently in use by the heap">
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-blue-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Heap In Use: {(memStats.heapinuse / (1024 * 1024)).toFixed(2)} MB</span>
                    </div>
                </Tooltip>
                <Tooltip content="Memory used by goroutine stacks">
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-green-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Stack: {(memStats.stackinuse / (1024 * 1024)).toFixed(2)} MB</span>
                    </div>
                </Tooltip>
                <Tooltip content="Memory used by memory spans, mcache, garbage collector, and other runtime allocations">
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-yellow-600 mr-1 rounded-sm"></div>
                        <span className="text-primary">Other: {((memStats.mspaninuse + memStats.mcacheinuse + memStats.gcsys + memStats.othersys) / (1024 * 1024)).toFixed(2)} MB</span>
                    </div>
                </Tooltip>
                <Tooltip content="Memory in the heap but not currently in use">
                    <div className="flex items-center cursor-pointer">
                        <div className="w-3 h-3 bg-gray-400 mr-1 rounded-sm"></div>
                        <span className="text-primary">Idle: {(memStats.heapidle / (1024 * 1024)).toFixed(2)} MB</span>
                    </div>
                </Tooltip>
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
}

const StatItem: React.FC<StatItemProps> = ({ label, value, unit }) => {
    return (
        <div className="mb-4 p-4 border border-border rounded-md bg-panel">
            <div className="text-sm text-secondary mb-1">{label}</div>
            <div className="text-2xl font-semibold text-primary">
                {value}
                {unit && <span className="text-sm text-secondary ml-1">{unit}</span>}
            </div>
        </div>
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
                <StatItem label="Memory Usage (Heap)" value={heapAllocMB} unit="MB" />
                <StatItem label="CPU Usage" value={stats.cpuusage.toFixed(2)} unit="%" />
                <StatItem label="Goroutine Count" value={stats.goroutinecount} />
                <StatItem label="Process ID" value={stats.pid} />
                <StatItem label="Working Directory" value={stats.cwd} />
                <StatItem label="GOMAXPROCS" value={stats.gomaxprocs} />
                <StatItem label="CPU Cores" value={stats.numcpu} />
                <StatItem label="Platform" value={`${stats.goos}/${stats.goarch}`} />
                <StatItem label="Go Version" value={stats.goversion} />
                
                {stats.memstats && (
                    <>
                        <StatItem 
                            label="Total Memory Allocated" 
                            value={totalAllocMB} 
                            unit="MB" 
                        />
                        <StatItem 
                            label="Total Process Memory" 
                            value={sysMB} 
                            unit="MB" 
                        />
                        <StatItem 
                            label="GC Cycles" 
                            value={stats.memstats.numgc} 
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
