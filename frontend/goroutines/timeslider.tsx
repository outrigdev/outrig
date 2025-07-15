// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { getDefaultStore, useAtomValue } from "jotai";
import { SkipForward } from "lucide-react";
import React from "react";
import { GoRoutinesModel } from "./goroutines-model";

interface TimeSliderProps {
    model: GoRoutinesModel;
}

export const TimeSlider: React.FC<TimeSliderProps> = ({ model }) => {
    const selectedTimestamp = useAtomValue(model.selectedTimestamp);
    const searchLatestMode = useAtomValue(model.searchLatestMode);
    const showActiveOnly = useAtomValue(model.showActiveOnly);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(model.appRunId);
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const fullTimeSpan = useAtomValue(model.fullTimeSpan);

    if (!appRunInfo || !fullTimeSpan) {
        return null;
    }

    // Calculate time range from fullTimeSpan
    const startTime = fullTimeSpan.start;
    const endTime = fullTimeSpan.end;
    const actualDurationSeconds = Math.floor((endTime - startTime) / 1000);
    const MAX_TIME_RANGE_SECONDS = 600 - 5;

    let maxRange: number;
    let adjustedStartTime: number;

    if (actualDurationSeconds <= MAX_TIME_RANGE_SECONDS) {
        maxRange = actualDurationSeconds;
        adjustedStartTime = startTime;
    } else {
        maxRange = MAX_TIME_RANGE_SECONDS;
        adjustedStartTime = endTime - MAX_TIME_RANGE_SECONDS * 1000;
    }

    // Don't render slider if no time range
    if (maxRange === 0) {
        return null;
    }

    // Convert timestamps to slider values (0 to maxRange seconds)
    // If selectedTimestamp is 0 or in search latest mode, push slider to the right
    const currentValue =
        searchLatestMode || selectedTimestamp === 0
            ? maxRange
            : Math.floor((selectedTimestamp - adjustedStartTime) / 1000);

    const handleSliderChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const sliderValue = parseInt(event.target.value);
        const timestamp = adjustedStartTime + sliderValue * 1000;
        model.setSelectedTimestampAndSearch(timestamp);
    };

    const handleSearchLatest = () => {
        model.enableSearchLatest();

        // Trigger a new search in latest mode
        const store = getDefaultStore();
        const searchTerm = store.get(model.searchTerm);
        model.searchGoroutines(searchTerm);
    };

    const formatTickLabel = (value: number, index: number, total: number): string => {
        const now = Date.now() / 1000;
        const isWithinCoupleSeconds = (time1: number, time2: number) => Math.abs(time1 - time2) <= 2;

        if (index === 0) {
            // First tick - check if it's the actual start
            if (isWithinCoupleSeconds(adjustedStartTime / 1000, appRunInfo.starttime)) {
                return "start";
            } else {
                // Drifted from start, use local timestamp
                const date = new Date(adjustedStartTime);
                return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
            }
        } else if (index === total - 1) {
            // Last tick - check if it's current/now
            const endTimeSeconds = endTime / 1000;
            if (
                isWithinCoupleSeconds(endTimeSeconds, now) ||
                isWithinCoupleSeconds(endTimeSeconds, appRunInfo.lastmodtime)
            ) {
                return "now";
            } else {
                // Drifted from current, use local timestamp
                const date = new Date(endTime);
                return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
            }
        } else {
            // Inner ticks - use relative seconds
            return `+${value}s`;
        }
    };

    // Calculate 5 tick marks: left, 3 internal, right
    const tickValues = [
        0,
        Math.floor(maxRange / 4),
        Math.floor(maxRange / 2),
        Math.floor((3 * maxRange) / 4),
        maxRange,
    ];
    const tickPositions = tickValues.map((value, index) => ({
        value,
        position: (value / maxRange) * 100,
        label: formatTickLabel(value, index, tickValues.length),
    }));

    return (
        <div className="flex items-center gap-2 flex-1 min-w-0 mt-[-8px]">
            <div className="flex items-center gap-2 shrink-0">
                <div className="flex items-center bg-muted rounded-md p-1">
                    <button
                        onClick={() => model.toggleShowActiveOnly()}
                        className={cn(
                            "px-2 py-1 text-xs rounded transition-colors",
                            !showActiveOnly
                                ? "bg-primary text-primary-foreground"
                                : "text-muted-foreground hover:text-primary"
                        )}
                    >
                        All
                    </button>
                    <button
                        onClick={() => model.toggleShowActiveOnly()}
                        className={cn(
                            "px-2 py-1 text-xs rounded transition-colors",
                            showActiveOnly
                                ? "bg-primary text-primary-foreground"
                                : "text-muted-foreground hover:text-primary"
                        )}
                    >
                        Active
                    </button>
                </div>
            </div>
            <div className="flex-1 relative">
                <input
                    type="range"
                    min="0"
                    max={maxRange}
                    step="1"
                    value={currentValue}
                    onChange={handleSliderChange}
                    className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer slider"
                    style={{
                        background: `linear-gradient(to right, var(--color-primary) 0%, var(--color-primary) ${(currentValue / maxRange) * 100}%, var(--color-muted) ${(currentValue / maxRange) * 100}%, var(--color-muted) 100%)`,
                    }}
                />
                {/* Tick marks with labels */}
                <div className="absolute top-3 left-0 right-0 pointer-events-none">
                    {tickPositions.map((tick, i) => {
                        const isFirst = i === 0;
                        const isLast = i === tickPositions.length - 1;
                        const alignmentClass = isFirst ? "items-start" : isLast ? "items-end" : "items-center";
                        const transformStyle = isFirst
                            ? "translateX(0%)"
                            : isLast
                              ? "translateX(-100%)"
                              : "translateX(-50%)";

                        return (
                            <div
                                key={i}
                                className={`absolute flex flex-col ${alignmentClass}`}
                                style={{ left: `${tick.position}%`, transform: transformStyle }}
                            >
                                <div className="w-px h-2 bg-primary" />
                                <span className="text-[10px] text-muted mt-[-1px] whitespace-nowrap">{tick.label}</span>
                            </div>
                        );
                    })}
                </div>
            </div>
            <Tooltip content={searchLatestMode ? "Search Latest (Active)" : "Search Latest"}>
                <button
                    onClick={handleSearchLatest}
                    className={cn(
                        "p-1 rounded transition-colors cursor-pointer",
                        searchLatestMode
                            ? "bg-primary/20 text-primary hover:bg-primary/30"
                            : "text-muted hover:bg-buttonhover hover:text-primary"
                    )}
                    aria-pressed={searchLatestMode ? "true" : "false"}
                >
                    <SkipForward size={14} />
                </button>
            </Tooltip>
        </div>
    );
};
