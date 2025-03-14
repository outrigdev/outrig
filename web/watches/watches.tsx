import { useAtom, useAtomValue } from "jotai";
import { Filter, RefreshCw } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { WatchesModel } from "./watches-model";

// Refresh button component
interface RefreshButtonProps {
    model: WatchesModel;
}

const RefreshButton: React.FC<RefreshButtonProps> = ({ model }) => {
    const isRefreshing = useAtomValue(model.isRefreshing);

    const handleRefresh = () => {
        model.refresh();
    };

    return (
        <button
            onClick={handleRefresh}
            className="p-1.5 border border-border rounded-md text-primary hover:bg-buttonhover transition-colors cursor-pointer"
            disabled={isRefreshing}
        >
            <RefreshCw size={14} className={isRefreshing ? "animate-spin" : ""} />
        </button>
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
        <div className="py-1 px-4 border-b border-border">
            <div className="flex items-center justify-between">
                <div className="flex items-center flex-grow">
                    <Filter
                        size={16}
                        className="text-muted mr-2"
                        fill="currentColor"
                        stroke="currentColor"
                        strokeWidth={1}
                    />
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
                <RefreshButton model={model} />
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
            ) : (
                <div className="flex items-center justify-center h-full text-secondary">
                    no watches found
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
    }, [model]);

    return (
        <div className="w-full h-full flex flex-col">
            <WatchesFilters model={model} />
            <WatchesContent model={model} />
        </div>
    );
};
