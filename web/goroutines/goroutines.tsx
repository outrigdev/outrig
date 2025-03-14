import { useAtom, useAtomValue } from "jotai";
import { Filter, RefreshCw } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { Tag } from "../elements/tag";
import { GoRoutinesModel } from "./goroutines-model";

// Individual goroutine view component
interface GoroutineViewProps {
    goroutine: GoroutineData;
}

const GoroutineView: React.FC<GoroutineViewProps> = ({ goroutine }) => {
    if (!goroutine) {
        return null;
    }

    return (
        <div className="mb-4 p-3 border border-border rounded-md hover:bg-buttonhover">
            <div className="flex justify-between items-center mb-2">
                <div className="font-semibold text-primary">Goroutine {goroutine.goid}</div>
                <div className="text-xs px-2 py-1 rounded-full bg-secondary/10 text-secondary">{goroutine.state}</div>
            </div>
            <pre className="text-xs text-primary overflow-auto whitespace-pre-wrap bg-panel p-2 rounded max-h-60">
                {goroutine.stacktrace}
            </pre>
        </div>
    );
};

// Refresh button component
interface RefreshButtonProps {
    model: GoRoutinesModel;
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

// Combined filters component for both search and state filters
interface GoRoutinesFiltersProps {
    model: GoRoutinesModel;
}

const GoRoutinesFilters: React.FC<GoRoutinesFiltersProps> = ({ model }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const [showAll, setShowAll] = useAtom(model.showAll);
    const [selectedStates, setSelectedStates] = useAtom(model.selectedStates);
    const availableStates = useAtomValue(model.availableStates);
    const searchRef = useRef<HTMLInputElement>(null);
    const isRefreshing = useAtomValue(model.isRefreshing);

    const handleToggleShowAll = () => {
        model.toggleShowAll();
    };

    const handleToggleState = (state: string) => {
        model.toggleStateFilter(state);
    };

    return (
        <>
            {/* Search filter */}
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
                            placeholder="Filter goroutines..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                    border-none ring-0 outline-none focus:outline-none focus:ring-0"
                        />
                    </div>
                    <RefreshButton model={model} />
                </div>
            </div>

            {/* Subtle divider */}
            <div className="h-px bg-border"></div>

            {/* State filters */}
            <div className="px-4 py-2 border-b border-border">
                <div className="flex flex-wrap items-center">
                    <Tag label="Show All" isSelected={showAll} onToggle={handleToggleShowAll} />
                    {availableStates.map((state) => (
                        <Tag
                            key={state}
                            label={state}
                            isSelected={selectedStates.has(state)}
                            onToggle={() => handleToggleState(state)}
                        />
                    ))}
                </div>
            </div>
        </>
    );
};

// Content component that displays the goroutines
interface GoRoutinesContentProps {
    model: GoRoutinesModel;
}

const GoRoutinesContent: React.FC<GoRoutinesContentProps> = ({ model }) => {
    const filteredGoroutines = useAtomValue(model.filteredGoroutines);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const search = useAtomValue(model.searchTerm);
    const showAll = useAtomValue(model.showAll);

    return (
        <div className="w-full h-full overflow-auto flex-1 p-4">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing goroutines...</span>
                    </div>
                </div>
            ) : filteredGoroutines.length === 0 ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search || !showAll ? "no goroutines match the filter" : "no goroutines found"}
                </div>
            ) : (
                <div>
                    <div className="mb-2 text-sm text-secondary">{filteredGoroutines.length} goroutines</div>
                    {filteredGoroutines.map((goroutine) => (
                        <GoroutineView key={goroutine.goid} goroutine={goroutine} />
                    ))}
                </div>
            )}
        </div>
    );
};

// Main goroutines component that composes the sub-components
interface GoRoutinesProps {
    appRunId: string;
}

export const GoRoutines: React.FC<GoRoutinesProps> = ({ appRunId }) => {
    const model = useRef(new GoRoutinesModel(appRunId)).current;

    useEffect(() => {
        // Load goroutines when the component mounts
        model.loadAppRunGoroutines();
    }, [model]);

    return (
        <div className="w-full h-full flex flex-col">
            <GoRoutinesFilters model={model} />
            <GoRoutinesContent model={model} />
        </div>
    );
};
