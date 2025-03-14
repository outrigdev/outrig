import { RefreshButton } from "@/elements/refreshbutton";
import { useOutrigModel } from "@/util/hooks";
import { useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { RuntimeStatsModel } from "./runtimestats-model";

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
            <RefreshButton
                isRefreshingAtom={model.isRefreshing}
                onRefresh={() => model.refresh()}
                tooltipContent="Refresh runtime stats"
                size={16}
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
    const formattedTime = new Date(stats.timestamp).toLocaleTimeString();

    return (
        <div className="w-full h-full overflow-auto p-4">
            <div className="text-sm text-secondary mb-4">Last updated: {formattedTime}</div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <StatItem label="Memory Usage" value={stats.memoryUsage.toFixed(2)} unit="MB" />
                <StatItem label="CPU Usage" value={stats.cpuUsage.toFixed(2)} unit="%" />
                <StatItem label="Goroutine Count" value={stats.goroutineCount} />
                {/* More stats can be added here when we hook up the backend */}
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
