import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { useOutrigModel } from "@/util/hooks";
import { cn } from "@/util/util";
import { PrimitiveAtom, useAtom, useAtomValue } from "jotai";
import { Filter, Layers, Layers2 } from "lucide-react";
import React, { useCallback, useRef } from "react";
import { Tag } from "../elements/tag";
import { GoRoutinesModel } from "./goroutines-model";
import { StackTrace } from "./stacktrace";

// StacktraceModeToggle component for toggling between raw and simplified stacktrace modes
interface StacktraceModeToggleProps {
    modeAtom: PrimitiveAtom<string>;
}

const StacktraceModeToggle: React.FC<StacktraceModeToggleProps> = ({ modeAtom }) => {
    const [mode, setMode] = useAtom(modeAtom);

    const handleToggleMode = useCallback(() => {
        // Cycle through the three modes: "raw" -> "simplified" -> "simplified:files" -> "raw"
        if (mode === "raw") {
            setMode("simplified");
        } else if (mode === "simplified") {
            setMode("simplified:files");
        } else {
            setMode("raw");
        }
    }, [mode, setMode]);

    // Determine tooltip content based on current mode
    const tooltipContent = useCallback(() => {
        switch (mode) {
            case "raw":
                return "Raw Stacktrace Mode (Click to Toggle)";
            case "simplified":
                return "Simplified Stacktrace Mode (Click to Toggle)";
            case "simplified:files":
                return "Simplified Stacktrace with Files Mode (Click to Toggle)";
            default:
                return "Toggle Stacktrace Mode";
        }
    }, [mode]);

    // Render the appropriate icon based on the current mode
    const renderIcon = useCallback(() => {
        switch (mode) {
            case "simplified":
                return <Layers2 size={16} />;
            case "simplified:files":
                return <Layers size={16} />;
            case "raw":
            default:
                return <Layers size={16} />;
        }
    }, [mode]);

    return (
        <Tooltip content={tooltipContent()}>
            <button
                onClick={handleToggleMode}
                className={cn(
                    "p-1 mr-1 rounded cursor-pointer transition-colors",
                    mode !== "raw"
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                )}
                aria-pressed={mode !== "raw" ? "true" : "false"}
            >
                {renderIcon()}
            </button>
        </Tooltip>
    );
};

// Individual goroutine view component
interface GoroutineViewProps {
    goroutine: ParsedGoRoutine;
    model: GoRoutinesModel;
}

const GoroutineView: React.FC<GoroutineViewProps> = ({ goroutine, model }) => {
    const linkType = useAtomValue(model.showCodeLinks);
    const simpleMode = useAtomValue(model.simpleStacktraceMode);
    const stackTraceRef = useRef<HTMLDivElement>(null);

    if (!goroutine) {
        return null;
    }

    const copyStackTrace = async () => {
        try {
            // If we have a ref to the stack trace div, use its text content
            if (stackTraceRef.current) {
                await navigator.clipboard.writeText(stackTraceRef.current.innerText);
            } else {
                // Fallback to raw stack trace if ref is not available
                await navigator.clipboard.writeText(goroutine.rawstacktrace);
            }
            return Promise.resolve();
        } catch (error) {
            console.error("Failed to copy stack trace:", error);
            return Promise.reject(error);
        }
    };

    return (
        <div className="mb-0">
            <div className="flex justify-between items-center py-2">
                <div className="flex items-center gap-2">
                    <div className="font-semibold text-primary w-[135px]">Goroutine {goroutine.goid}</div>
                    <div className="text-xs px-2 py-1 rounded-full bg-secondary/10 text-secondary">
                        {goroutine.rawstate}
                    </div>
                </div>
                <div>
                    <CopyButton
                        onCopy={copyStackTrace}
                        tooltipText="Copy stack trace"
                        successTooltipText="Stack trace copied!"
                        size={14}
                    />
                </div>
            </div>
            <div ref={stackTraceRef} className="pb-2">
                <StackTrace goroutine={goroutine} model={model} linkType={linkType} simpleMode={simpleMode} />
            </div>
        </div>
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
    const [simpleMode, setSimpleMode] = useAtom(model.simpleStacktraceMode);
    const availableStates = useAtomValue(model.availableStates);
    const searchRef = useRef<HTMLInputElement>(null);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const filteredCount = useAtomValue(model.filteredCount);
    const totalCount = useAtomValue(model.totalCount);

    const handleToggleShowAll = () => {
        model.toggleShowAll();
    };

    const handleToggleState = (state: string) => {
        model.toggleStateFilter(state);
    };

    return (
        <>
            {/* Search filter */}
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
                            placeholder="Filter goroutines..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
                        />
                    </div>

                    {/* Search stats */}
                    <div className="text-xs text-muted mr-2 select-none">
                        {filteredCount}/{totalCount}
                    </div>

                    <div className="flex items-center gap-2">
                        <StacktraceModeToggle modeAtom={model.simpleStacktraceMode} />
                        <RefreshButton
                            isRefreshingAtom={model.isRefreshing}
                            onRefresh={() => model.refresh()}
                            tooltipContent="Refresh goroutines"
                            size={16}
                        />
                    </div>
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
                    {filteredGoroutines.map((goroutine, index) => (
                        <React.Fragment key={goroutine.goid}>
                            <GoroutineView goroutine={goroutine} model={model} />
                            {/* Add divider after each goroutine except the last one */}
                            {index < filteredGoroutines.length - 1 && (
                                <div
                                    className="h-px bg-border my-2"
                                    style={{ minWidth: "100%", width: "9999px" }}
                                ></div>
                            )}
                        </React.Fragment>
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
    const model = useOutrigModel(GoRoutinesModel, appRunId);

    console.log("Render goroutines", appRunId, model);

    if (!model) {
        return null;
    }

    return (
        <div className="w-full h-full flex flex-col">
            <GoRoutinesFilters model={model} />
            <GoRoutinesContent model={model} />
        </div>
    );
};
