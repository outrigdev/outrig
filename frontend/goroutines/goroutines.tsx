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
import { Layers, Layers2, Search } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { Tag } from "../elements/tag";
import { GoRoutinesModel } from "./goroutines-model";
import { StackTrace } from "./stacktrace";

// StacktraceModeToggle component for toggling between raw and simplified stacktrace modes
interface StacktraceModeToggleProps {
    modeAtom: PrimitiveAtom<string>;
    model: GoRoutinesModel;
}

const StacktraceModeToggle: React.FC<StacktraceModeToggleProps> = ({ modeAtom, model }) => {
    const [mode, setMode] = useAtom(modeAtom);
    const searchTerm = useAtomValue(model.searchTerm);
    const isSearchActive = searchTerm && searchTerm.trim() !== "";

    const handleToggleMode = useCallback(() => {
        // If search is active, don't allow toggling
        if (isSearchActive) return;

        // Cycle through the three modes: "raw" -> "simplified" -> "simplified:files" -> "raw"
        if (mode === "raw") {
            setMode("simplified");
        } else if (mode === "simplified") {
            setMode("simplified:files");
        } else {
            setMode("raw");
        }
    }, [mode, setMode, isSearchActive]);

    // Determine tooltip content based on current mode and search state
    const tooltipContent = useCallback(() => {
        if (isSearchActive) {
            return "Raw Stacktrace Mode Locked (to reveal search matches)";
        }

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
    }, [mode, isSearchActive]);

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
                    "p-1 mr-1 rounded transition-colors relative",
                    isSearchActive ? "cursor-default" : "cursor-pointer",
                    mode !== "raw"
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                )}
                aria-pressed={mode !== "raw" ? "true" : "false"}
            >
                {renderIcon()}

                {/* Show search indicator when search is active */}
                {isSearchActive && (
                    <div className="absolute -top-1 -right-1 bg-accent rounded-full p-0.5">
                        <Search size={10} className="text-white" />
                    </div>
                )}
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
    const appRunStartTime = useAtomValue(AppModel.appRunStartTimeAtom);
    // Use the effective mode which automatically switches to "raw" when search is active
    const simpleMode = useAtomValue(model.effectiveSimpleStacktraceMode);
    const stackTraceRef = useRef<HTMLDivElement>(null);

    if (!goroutine) {
        return null;
    }

    const copyStackTrace = async () => {
        try {
            await navigator.clipboard.writeText(goroutine.rawstacktrace);
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
                    <div className="text-primary whitespace-nowrap overflow-hidden text-ellipsis pr-4 flex items-center">
                        <span className="font-semibold">
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
                                <span className="ml-2 text-xs text-muted bg-muted/10 px-1 py-0.5 rounded hover:bg-muted/20 transition-colors font-semibold">
                                    +{formatTimeOffset(goroutine.firstseen, appRunStartTime)}
                                </span>
                            </Tooltip>
                        )}
                        {/* Display state */}
                        {goroutine.primarystate && (
                            <Tag
                                key="primary-state"
                                label={
                                    goroutine.stateduration
                                        ? `${goroutine.primarystate} (${goroutine.stateduration})`
                                        : goroutine.primarystate
                                }
                                isSelected={false}
                                variant="secondary"
                                className="ml-2"
                            />
                        )}
                    </div>
                    <div className="flex items-center gap-1">
                        <CopyButton
                            onCopy={copyStackTrace}
                            tooltipText="Copy stack trace"
                            successTooltipText="Stack trace copied!"
                            size={14}
                        />
                    </div>
                </div>
                {/* Only show tags row if there are tags */}
                {goroutine.tags && goroutine.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                        {/* Display tags with # prefix */}
                        {goroutine.tags.map((tag, index) => (
                            <Tag key={`tag-${index}`} label={`#${tag}`} isSelected={false} variant="accent" />
                        ))}
                    </div>
                )}
            </div>
            <div ref={stackTraceRef} className="pb-2">
                <StackTrace goroutine={goroutine} model={model} simpleMode={simpleMode} />
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

                    <Tooltip content="Matched GoRoutines / Total GoRoutine Count">
                        <div className="text-xs text-muted mr-2 cursor-default">
                            <span>
                                {resultCount}/{searchResultInfo.totalCount}
                            </span>
                        </div>
                    </Tooltip>

                    <div className="flex items-center gap-2">
                        <StacktraceModeToggle modeAtom={model.simpleStacktraceMode} model={model} />
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
                    <div className="flex items-start shrink-0">
                        <Tag label="Show All" isSelected={showAll} onToggle={handleToggleShowAll} />
                    </div>

                    {/* Box 2: Primary States */}
                    <div className="flex-grow flex flex-wrap items-start gap-1.5">
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

                    {/* Box 3: #outrig toggle */}
                    <div className="flex items-start shrink-0">
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
    const sortedGoroutines = useAtomValue(model.sortedGoRoutines);
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
        if (sortedGoroutines.length === 0 && !isRefreshing) {
            const timer = setTimeout(() => {
                setShowEmptyMessage(true);
            }, EmptyMessageDelayMs);

            return () => clearTimeout(timer);
        } else {
            setShowEmptyMessage(false);
        }
    }, [sortedGoroutines.length, isRefreshing]);

    return (
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing goroutines...</span>
                    </div>
                </div>
            ) : sortedGoroutines.length === 0 && showEmptyMessage ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search || !showAll ? "no goroutines match the filter" : "no goroutines found"}
                </div>
            ) : (
                <div>
                    {sortedGoroutines.map((goroutine, index) => (
                        <React.Fragment key={goroutine.goid}>
                            <GoroutineView goroutine={goroutine} model={model} />
                            {/* Add divider after each goroutine except the last one */}
                            {index < sortedGoroutines.length - 1 && <div className="h-px bg-border my-2 w-full"></div>}
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
