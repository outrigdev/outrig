import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { useOutrigModel } from "@/util/hooks";
import { cn } from "@/util/util";
import { useAtom, useAtomValue } from "jotai";
import { Filter, Layers } from "lucide-react";
import React, { useRef } from "react";
import { Tag } from "../elements/tag";
import { CodeLinkType, GoRoutinesModel } from "./goroutines-model";
import { simplifyStackTrace } from "./stacktrace";

// Individual goroutine view component
interface GoroutineViewProps {
    goroutine: GoroutineData;
    model: GoRoutinesModel;
}

// Component for a single stacktrace line with optional VSCode link
interface StacktraceLineProps {
    line: string;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
}

const StacktraceLine: React.FC<StacktraceLineProps> = ({ line, model, linkType }) => {
    // Only process lines that might contain file paths
    if (!line.includes(".go:")) {
        return <div>{line}</div>;
    }

    const parsedLine = model.parseStacktraceLine(line);
    if (!parsedLine || linkType == null) {
        return <div>{line}</div>;
    }

    const { filePath, lineNumber } = parsedLine;
    const link = model.generateCodeLink(filePath, lineNumber, linkType);

    if (!link) {
        return <div>{line}</div>;
    }

    // Find the file:line part in the text to make it clickable
    const fileLinePattern = new RegExp(`(${filePath.replace(/\//g, "\\/")}:${lineNumber})`);
    const parts = line.split(fileLinePattern);

    if (parts.length === 1) {
        // Pattern not found, return the line as is
        return <div>{line}</div>;
    }

    return (
        <div>
            {parts.map((part, index) => {
                // If this part matches the file:line pattern, make it a link
                if (part === `${filePath}:${lineNumber}`) {
                    return (
                        <a key={index} href={link} className="group cursor-pointer">
                            <span className="group-hover:text-blue-500 group-hover:underline transition-colors duration-150">
                                {part}
                            </span>
                        </a>
                    );
                }
                return <span key={index}>{part}</span>;
            })}
        </div>
    );
};

const GoroutineView: React.FC<GoroutineViewProps> = ({ goroutine, model }) => {
    const linkType = useAtomValue(model.showCodeLinks);
    const simpleMode = useAtomValue(model.simpleStacktraceMode);

    if (!goroutine) {
        return null;
    }

    // Apply simplification if simple mode is enabled
    const displayStacktrace = simpleMode ? simplifyStackTrace(goroutine.stacktrace) : goroutine.stacktrace;

    // Split the stacktrace into lines
    const stacktraceLines = displayStacktrace.split("\n");

    const copyStackTrace = async () => {
        try {
            await navigator.clipboard.writeText(goroutine.stacktrace);
            return Promise.resolve();
        } catch (error) {
            console.error("Failed to copy stack trace:", error);
            return Promise.reject(error);
        }
    };

    return (
        <div className="mb-4 p-3 border border-border rounded-md">
            <div className="flex justify-between items-center mb-2">
                <div className="flex items-center gap-2">
                    <div className="font-semibold text-primary w-[135px]">Goroutine {goroutine.goid}</div>
                    <div className="text-xs px-2 py-1 rounded-full bg-secondary/10 text-secondary">{goroutine.state}</div>
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
            <pre className="text-xs text-primary whitespace-pre-wrap bg-panel p-2 rounded">
                {stacktraceLines.map((line, index) => (
                    <StacktraceLine key={index} line={line} model={model} linkType={linkType} />
                ))}
            </pre>
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
                    <div className="flex items-center gap-2">
                        <Tooltip
                            content={
                                simpleMode
                                    ? "Simple Stacktrace Mode On (Click to Disable)"
                                    : "Simple Stacktrace Mode Off (Click to Enable)"
                            }
                        >
                            <button
                                onClick={() => setSimpleMode(!simpleMode)}
                                className={cn(
                                    "p-1 mr-1 rounded cursor-pointer transition-colors",
                                    simpleMode
                                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                                        : "text-muted hover:bg-buttonhover hover:text-primary"
                                )}
                                aria-pressed={simpleMode}
                            >
                                <Layers size={16} />
                            </button>
                        </Tooltip>
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
                    <div className="mb-2 text-sm text-secondary">{filteredGoroutines.length} goroutines</div>
                    {filteredGoroutines.map((goroutine) => (
                        <GoroutineView key={goroutine.goid} goroutine={goroutine} model={model} />
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
