import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { useAtom, useAtomValue } from "jotai";
import { Clock, Filter, Timer } from "lucide-react";
import React, { useCallback, useEffect, useRef } from "react";
import { WatchesModel } from "./watches-model";

// Individual watch view component
interface WatchViewProps {
    watch: Watch;
}

const WatchView: React.FC<WatchViewProps> = ({ watch }) => {
    if (!watch) {
        return null;
    }

    // Format the watch value for display
    const formatValue = (watch: Watch) => {
        if (watch.error) {
            return <span className="text-error">{watch.error}</span>;
        }

        if (watch.value == null) {
            return <span className="text-muted">null</span>;
        }

        // Try to parse JSON if it looks like JSON
        if (watch.value.startsWith("{") || watch.value.startsWith("[")) {
            try {
                const parsed = JSON.parse(watch.value);
                return <pre className="text-xs whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>;
            } catch {
                // If it's not valid JSON, just display as is
            }
        }

        return <span>{watch.value}</span>;
    };

    return (
        <div className="mb-4 p-3 border border-border rounded-md hover:bg-buttonhover">
            <div className="flex justify-between items-center mb-2">
                <div className="font-semibold text-primary">{watch.name}</div>
                <div className="flex items-center gap-2">
                    <div className="text-xs px-2 py-1 rounded-full bg-secondary/10 text-secondary">{watch.type}</div>
                    <CopyButton
                        size={14}
                        tooltipText="Copy value"
                        onCopy={() => {
                            if (watch.value) {
                                navigator.clipboard.writeText(watch.value);
                            }
                        }}
                    />
                </div>
            </div>
            <div className="text-sm text-primary bg-panel p-2 rounded">{formatValue(watch)}</div>
            {(watch.len != null || watch.cap != null) && (
                <div className="mt-2 text-xs text-muted">
                    {watch.len != null && <span>Length: {watch.len}</span>}
                    {watch.len != null && watch.cap != null && <span> | </span>}
                    {watch.cap != null && <span>Capacity: {watch.cap}</span>}
                </div>
            )}
            {watch.waittime != null && watch.waittime > 0 && (
                <div className="mt-1 text-xs text-warning">Wait time: {watch.waittime}μs</div>
            )}
        </div>
    );
};

// Auto-refresh button component
interface AutoRefreshButtonProps {
    model: WatchesModel;
}

const AutoRefreshButton: React.FC<AutoRefreshButtonProps> = ({ model }) => {
    const [autoRefresh, setAutoRefresh] = useAtom(model.autoRefresh);

    const toggleAutoRefresh = useCallback(() => {
        model.toggleAutoRefresh();
    }, [model]);

    return (
        <Tooltip content={autoRefresh ? "Auto-refresh On (Click to Disable)" : "Auto-refresh Off (Click to Enable)"}>
            <button
                onClick={toggleAutoRefresh}
                className={`p-1 mr-1 rounded ${
                    autoRefresh
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                } cursor-pointer transition-colors`}
                aria-pressed={autoRefresh}
            >
                {autoRefresh ? <Timer size={16} /> : <Clock size={16} />}
            </button>
        </Tooltip>
    );
};

// Watches filters component
interface WatchesFiltersProps {
    model: WatchesModel;
}

const WatchesFilters: React.FC<WatchesFiltersProps> = ({ model }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const searchRef = useRef<HTMLInputElement>(null);

    return (
        <div className="py-1 px-1 border-b border-border">
            <div className="flex items-center justify-between">
                <div className="flex items-center flex-grow">
                    <div className="select-none pr-2 text-muted w-12 text-right font-mono flex justify-end items-center">
                        <Filter
                            size={16}
                            className="text-muted"
                            fill="currentColor"
                            stroke="currentColor"
                            strokeWidth={1}
                        />
                    </div>
                    <input
                        ref={searchRef}
                        type="text"
                        placeholder="Filter watches..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
                    />
                </div>
                <AutoRefreshButton model={model} />
                <RefreshButton
                    isRefreshingAtom={model.isRefreshing}
                    onRefresh={() => model.refresh()}
                    tooltipContent="Refresh watches"
                    size={16}
                />
            </div>
        </div>
    );
};

// Content component that displays the watches
interface WatchesContentProps {
    model: WatchesModel;
}

const WatchesContent: React.FC<WatchesContentProps> = ({ model }) => {
    const filteredWatches = useAtomValue(model.filteredWatches);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const search = useAtomValue(model.searchTerm);

    return (
        <div className="w-full h-full overflow-auto flex-1 p-4">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing watches...</span>
                    </div>
                </div>
            ) : filteredWatches.length === 0 ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search ? "no watches match the filter" : "no watches found"}
                </div>
            ) : (
                <div>
                    <div className="mb-2 text-sm text-secondary">{filteredWatches.length} watches</div>
                    {filteredWatches.map((watch) => (
                        <WatchView key={watch.name} watch={watch} />
                    ))}
                </div>
            )}
        </div>
    );
};

// Main watches component that composes the sub-components
interface WatchesProps {
    appRunId: string;
}

export const Watches: React.FC<WatchesProps> = ({ appRunId }) => {
    const model = useRef(new WatchesModel(appRunId)).current;

    useEffect(() => {
        // Initialize when the component mounts
        model.refresh();
        
        // Ensure auto-refresh is started (in case it wasn't started in the constructor)
        model.startAutoRefreshInterval();
        
        // Clean up when the component unmounts
        return () => {
            model.dispose();
        };
    }, [model]);

    return (
        <div className="w-full h-full flex flex-col">
            <WatchesFilters model={model} />
            <WatchesContent model={model} />
        </div>
    );
};
