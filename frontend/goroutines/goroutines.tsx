// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { SearchFilter } from "@/searchfilter/searchfilter";
import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { checkKeyPressed } from "@/util/keyutil";
import { cn, formatTimeOffset } from "@/util/util";
import { PrimitiveAtom, useAtom, useAtomValue } from "jotai";
import { Layers, Layers2 } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { Tag } from "../elements/tag";
import { GoRoutinesModel } from "./goroutines-model";
import { StackTrace } from "./stacktrace";

// Duration state filters component
interface DurationStateFiltersProps {
    model: GoRoutinesModel;
    selectedStates: Set<string>;
    onToggleState: (state: string) => void;
}

const DurationStateFilters: React.FC<DurationStateFiltersProps> = ({ model, selectedStates, onToggleState }) => {
    const durationStates = useAtomValue(model.durationStates);
    const stateCounts = useAtomValue(model.stateCounts);

    if (durationStates.length === 0) {
        return null;
    }

    return (
        <div className="flex flex-wrap items-start gap-1.5">
            {durationStates.map((state) => (
                <Tag
                    key={state}
                    label={state}
                    count={stateCounts.get(state) || 0}
                    isSelected={selectedStates.has(state)}
                    onToggle={() => onToggleState(state)}
                />
            ))}
        </div>
    );
};

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
    const appRunStartTime = useAtomValue(AppModel.appRunStartTimeAtom);
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
        <div className="pl-4 pr-2">
            <div className="py-2">
                <div className="flex justify-between items-center">
                    <div className="font-semibold text-primary whitespace-nowrap overflow-hidden text-ellipsis pr-4 flex items-center">
                        <span>
                            {goroutine.name ? (
                                <>
                                    {goroutine.name} <span className="text-secondary">({goroutine.goid})</span>
                                </>
                            ) : (
                                `Goroutine ${goroutine.goid}`
                            )}
                        </span>
                        {goroutine.firstseen && appRunStartTime && (
                            <Tooltip content={`Goroutine started at ${new Date(goroutine.firstseen).toLocaleString()}`}>
                                <span className="ml-2 text-xs text-muted bg-muted/10 px-1 py-0.5 rounded hover:bg-muted/20 transition-colors">
                                    {formatTimeOffset(goroutine.firstseen, appRunStartTime)}
                                </span>
                            </Tooltip>
                        )}
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
                <div className="flex flex-wrap gap-1 mt-1">
                    {/* Display states */}
                    {goroutine.rawstate.split(",").map((state, index) => (
                        <Tag key={`state-${index}`} label={state.trim()} isSelected={false} variant="secondary" />
                    ))}

                    {/* Display tags with # prefix if they exist */}
                    {goroutine.tags &&
                        goroutine.tags.length > 0 &&
                        goroutine.tags.map((tag, index) => (
                            <Tag key={`tag-${index}`} label={`#${tag}`} isSelected={false} variant="accent" />
                        ))}
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
    const [showOutrig, setShowOutrig] = useAtom(model.showOutrigGoroutines);
    const [selectedStates, setSelectedStates] = useAtom(model.selectedStates);
    const searchResultInfo = useAtomValue(model.searchResultInfo);
    const resultCount = useAtomValue(model.resultCount);
    const primaryStates = useAtomValue(model.primaryStates);
    const stateCounts = useAtomValue(model.stateCounts);
    const errorSpans = searchResultInfo.errorSpans || [];

    const handleToggleShowAll = () => {
        model.toggleShowAll();
    };

    const handleToggleShowOutrig = () => {
        model.toggleShowOutrigGoroutines();
    };

    const handleToggleState = (state: string) => {
        model.toggleStateFilter(state);
    };

    return (
        <>
            {/* Search filter */}
            <div className="py-1 px-1 border-b border-border">
                <div className="flex items-center justify-between">
                    <SearchFilter
                        value={search}
                        onValueChange={(value) => {
                            setSearch(value);
                            model.updateSearchTerm(value);
                        }}
                        placeholder="Filter goroutines..."
                        autoFocus={true}
                        errorSpans={errorSpans}
                        onOutrigKeyDown={(keyEvent) => {
                            if (checkKeyPressed(keyEvent, "PageUp")) {
                                model.pageUp();
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "PageDown")) {
                                model.pageDown();
                                return true;
                            }
                            return false;
                        }}
                    />

                    {/* Search stats */}
                    <div className="text-xs text-muted mr-2 select-none">
                        <span>
                            {resultCount}/{searchResultInfo.totalCount}
                        </span>
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
                <div className="flex items-start gap-x-2">
                    {/* Box 1: Show All */}
                    <div className="shrink-0">
                        <Tag label="Show All" isSelected={showAll} onToggle={handleToggleShowAll} />
                    </div>

                    {/* Box 2: Primary and Extra states in a flex column with flex-grow */}
                    <div className="flex-grow flex flex-col gap-y-1">
                        <div className="flex flex-wrap items-start gap-1.5">
                            {/* Primary states first */}
                            {primaryStates.map((state) => (
                                <Tag
                                    key={state}
                                    label={state}
                                    count={stateCounts.get(state) || 0}
                                    isSelected={selectedStates.has(state)}
                                    onToggle={() => handleToggleState(state)}
                                />
                            ))}
                        </div>
                    </div>

                    {/* Box 3: #outrig toggle */}
                    <div className="shrink-0">
                        <Tooltip
                            content={
                                showOutrig
                                    ? "Showing Internal Outrig SDK GoRoutines (Click to Toggle)"
                                    : "Hiding Internal Outrig SDK GoRoutines (Click to Toggle)"
                            }
                        >
                            <div>
                                <Tag
                                    label="#outrig"
                                    isSelected={showOutrig}
                                    onToggle={handleToggleShowOutrig}
                                    variant="accent"
                                />
                            </div>
                        </Tooltip>
                    </div>
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
    const goroutines = useAtomValue(model.appRunGoRoutines);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const search = useAtomValue(model.searchTerm);
    const showAll = useAtomValue(model.showAll);
    const contentRef = useRef<HTMLDivElement>(null);
    const [showEmptyMessage, setShowEmptyMessage] = useState(false);

    // Set the content ref in the model when it changes
    useEffect(() => {
        model.setContentRef(contentRef);
    }, [model]);

    // Set a timeout to show empty message after component mounts or when goroutines change
    useEffect(() => {
        if (goroutines.length === 0 && !isRefreshing) {
            const timer = setTimeout(() => {
                setShowEmptyMessage(true);
            }, EmptyMessageDelayMs);

            return () => clearTimeout(timer);
        } else {
            setShowEmptyMessage(false);
        }
    }, [goroutines.length, isRefreshing]);

    return (
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing goroutines...</span>
                    </div>
                </div>
            ) : goroutines.length === 0 && showEmptyMessage ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search || !showAll ? "no goroutines match the filter" : "no goroutines found"}
                </div>
            ) : (
                <div>
                    {goroutines.map((goroutine, index) => (
                        <React.Fragment key={goroutine.goid}>
                            <GoroutineView goroutine={goroutine} model={model} />
                            {/* Add divider after each goroutine except the last one */}
                            {index < goroutines.length - 1 && (
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
