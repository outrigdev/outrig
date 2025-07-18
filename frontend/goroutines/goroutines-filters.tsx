// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { RefreshButton } from "@/elements/refreshbutton";
import { Tooltip } from "@/elements/tooltip";
import { SearchFilter } from "@/searchfilter/searchfilter";
import { checkKeyPressed } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import React from "react";
import { Tag } from "../elements/tag";
import { GoRoutineTimelineScrubber } from "./goroutine-timeline-scrubber";
import { GoRoutinesModel } from "./goroutines-model";
import { SearchLatestButton } from "./search-latest-button";
import { StacktraceModeToggle } from "./stacktrace-mode-toggle";

interface GoRoutinesFiltersProps {
    model: GoRoutinesModel;
}

export const GoRoutinesFilters: React.FC<GoRoutinesFiltersProps> = ({ model }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const [showAll, setShowAll] = useAtom(model.showAll);
    const [showOutrig, setShowOutrig] = useAtom(model.showOutrigGoroutines);
    const [selectedStates, setSelectedStates] = useAtom(model.selectedStates);
    const [showActiveOnly, setShowActiveOnly] = useAtom(model.showActiveOnly);
    const searchResultInfo = useAtomValue(model.searchResultInfo);
    const resultCount = useAtomValue(model.resultCount);
    const lastSearchTimestamp = useAtomValue(model.lastSearchTimestamp);
    const timeOffsetSeconds = useAtomValue(model.timeOffsetSeconds);
    const errorSpans = searchResultInfo.errorSpans || [];

    const goroutineStateCounts = searchResultInfo.goroutinestatecounts || {};

    const availableStates = Object.keys(goroutineStateCounts)
        .filter((state) => state !== "inactive")
        .sort();

    const activeCount = Object.entries(goroutineStateCounts)
        .filter(([state]) => state !== "inactive")
        .reduce((sum, [, count]) => sum + count, 0);

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

                    <StacktraceModeToggle modeAtom={model.simpleStacktraceMode} model={model} />

                    <div className="flex items-center gap-2">
                        <RefreshButton
                            isRefreshingAtom={model.isRefreshing}
                            onRefresh={() => model.refresh()}
                            tooltipContent="Refresh goroutines"
                            size={16}
                        />
                    </div>
                </div>
            </div>

            {lastSearchTimestamp > 0 && (
                <div className="px-4 py-2 space-y-3">
                    <div className="flex gap-3">
                        <div className="flex-1">
                            <GoRoutineTimelineScrubber model={model} />
                        </div>
                        <div className="flex flex-col">
                            <div className="text-xs text-muted font-medium h-[16px]"></div>
                            <div className="flex items-center h-8 mt-4">
                                <SearchLatestButton model={model} />
                            </div>
                        </div>
                    </div>
                </div>
            )}

            <div className="px-4 py-2 border-b border-border mt-1">
                <div className="flex items-start gap-x-2">
                    <div className="flex items-start shrink-0">
                        <Tag
                            label="All"
                            count={showOutrig ? searchResultInfo.totalCount : searchResultInfo.totalnonoutrig}
                            isSelected={showAll && selectedStates.size === 0 && !showActiveOnly}
                            onToggle={handleToggleShowAll}
                        />
                    </div>

                    <div className="flex-grow flex flex-wrap items-start gap-1.5">
                        {activeCount > 0 && (
                            <Tag
                                key="active"
                                label="Active"
                                count={activeCount}
                                isSelected={showActiveOnly}
                                onToggle={() => model.toggleShowActiveOnly()}
                            />
                        )}
                        {availableStates.map((state) => (
                            <Tag
                                key={state}
                                label={state}
                                count={goroutineStateCounts[state] || 0}
                                isSelected={selectedStates.has(state)}
                                onToggle={() => handleToggleState(state)}
                            />
                        ))}
                    </div>

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
                                    variant={showOutrig ? "accent" : "secondary"}
                                />
                            </div>
                        </Tooltip>
                    </div>
                </div>
            </div>
        </>
    );
};
