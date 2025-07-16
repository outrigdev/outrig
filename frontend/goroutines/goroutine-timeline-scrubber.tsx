// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import React from "react";
import { GoRoutinesModel } from "./goroutines-model";

const Debug = false;
const GraphHeight = 40;
const GridPercentages = [25, 50, 75];
const TimeFormatOptions: Intl.DateTimeFormatOptions = {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
};

// Helper function to format time consistently
function formatTime(timestamp: number): string {
    return new Date(timestamp).toLocaleTimeString([], TimeFormatOptions);
}

// Helper function to create SVG path for goroutine timeline with proper null handling
function createGoRoutineTimelinePath(
    activeCounts: GoRoutineActiveCount[],
    minTimeIdx: number,
    plotTimeIdxRange: number,
    maxCount: number
): string {
    if (activeCounts.length === 0) return "";

    const graphHeight = GraphHeight;
    let path = "";
    let currentPath = "";
    let inPath = false;

    // Create a map for quick lookup of counts by timeidx
    const countMap = new Map<number, number>();
    activeCounts.forEach((count) => {
        countMap.set(count.timeidx, count.count);
    });

    // Iterate through the entire timeline range
    const startTimeIdx = minTimeIdx;
    const endTimeIdx = minTimeIdx + plotTimeIdxRange;

    for (let timeIdx = startTimeIdx; timeIdx <= endTimeIdx; timeIdx++) {
        const count = countMap.get(timeIdx);
        const x = ((timeIdx - minTimeIdx) / plotTimeIdxRange) * 100;

        if (count !== undefined) {
            // We have data for this point
            const y = graphHeight - (count / maxCount) * graphHeight;

            if (!inPath) {
                // Start a new path segment
                currentPath = `M ${x} ${graphHeight} L ${x} ${y}`;
                inPath = true;
            } else {
                // Continue the current path
                currentPath += ` L ${x} ${y}`;
            }
        } else {
            // No data for this point (null) - close current path if we have one
            if (inPath) {
                // Close the current path segment
                const lastX = ((timeIdx - 1 - minTimeIdx) / plotTimeIdxRange) * 100;
                currentPath += ` L ${lastX} ${graphHeight} Z`;
                path += currentPath;
                currentPath = "";
                inPath = false;
            }
        }
    }

    // Close any remaining open path
    if (inPath) {
        const lastX = ((endTimeIdx - minTimeIdx) / plotTimeIdxRange) * 100;
        currentPath += ` L ${lastX} ${graphHeight} Z`;
        path += currentPath;
    }

    return path;
}

interface GoRoutineTimelineScrubberProps {
    model: GoRoutinesModel;
}

export const GoRoutineTimelineScrubber: React.FC<GoRoutineTimelineScrubberProps> = ({ model }) => {
    const activeCounts = useAtomValue(model.activeCounts);
    const selectedTimestamp = useAtomValue(model.selectedTimestamp);
    const searchLatestMode = useAtomValue(model.searchLatestMode);
    const timelineRange = useAtomValue(model.timelineRangeAtom);
    const fullTimeSpan = useAtomValue(model.fullTimeSpan);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(model.appRunId);
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const isAppRunning = useAtomValue(AppModel.selectedAppRunIsRunningAtom);

    if (!appRunInfo || !fullTimeSpan || activeCounts.length === 0) {
        return null;
    }

    // Sort active counts by timeidx to ensure proper ordering
    const sortedActiveCounts = [...activeCounts].sort((a, b) => a.timeidx - b.timeidx);

    // Extract values from timeline range
    const { minTimeIdx, maxTimeIdx } = timelineRange;
    const baseTimeIdxRange = maxTimeIdx - minTimeIdx;
    if (baseTimeIdxRange === 0) {
        return null;
    }

    // Find max count for scaling
    const maxCount = Math.max(...sortedActiveCounts.map((c) => c.count));
    if (maxCount === 0) {
        return null;
    }

    // Calculate the range to use for plotting (padded when running, actual when stopped)
    const plotTimeIdxRange = isAppRunning ? timelineRange.paddedMaxTimeIdx - minTimeIdx : baseTimeIdxRange;

    // Create a map for efficient lookups
    const timeIdxToCountMap = new Map(sortedActiveCounts.map((count) => [count.timeidx, count]));

    // Helper function to convert timeidx to timestamp
    const timeIdxToTimestamp = (timeIdx: number): number => {
        const activeCount = timeIdxToCountMap.get(timeIdx);
        return activeCount?.tsts ?? 0;
    };

    // Helper function to find the closest timeidx for a given timestamp
    const timestampToTimeIdx = (timestamp: number): number => {
        let closestIdx = minTimeIdx;
        let closestDiff = Infinity;

        for (const count of sortedActiveCounts) {
            const diff = Math.abs(count.tsts - timestamp);
            if (diff < closestDiff) {
                closestDiff = diff;
                closestIdx = count.timeidx;
            }
        }

        return closestIdx;
    };

    // Calculate current slider value based on selected timestamp
    const currentTimeIdx = searchLatestMode ? maxTimeIdx : timestampToTimeIdx(selectedTimestamp);

    const handleSliderChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const timeIdx = parseInt(event.target.value);
        const timestamp = timeIdxToTimestamp(timeIdx);
        if (timestamp > 0) {
            model.setSelectedTimestampAndSearch(timestamp);
        }
    };

    const handleGraphClick = (event: React.MouseEvent<HTMLDivElement>) => {
        const rect = event.currentTarget.getBoundingClientRect();
        const clickX = event.clientX - rect.left;
        const graphWidth = rect.width;
        const clickPercent = clickX / graphWidth;

        // Convert click position to timeidx using the plot range
        const clickedTimeIdx = Math.round(minTimeIdx + clickPercent * plotTimeIdxRange);
        const timestamp = timeIdxToTimestamp(clickedTimeIdx);

        if (timestamp > 0) {
            model.setSelectedTimestampAndSearch(timestamp);
        }
    };

    // Calculate slider position marker
    // Only consider it at the last position if we're at the actual end of the plot range
    const plotMaxTimeIdx = isAppRunning ? timelineRange.paddedMaxTimeIdx : maxTimeIdx;
    const isAtLastPosition = currentTimeIdx === plotMaxTimeIdx;
    const sliderMarkerPercent = ((currentTimeIdx - minTimeIdx) / plotTimeIdxRange) * 100;

    return (
        <div className="flex flex-col gap-2">
            <div className="text-xs text-muted font-medium">Active Goroutines</div>
            {/* Debug info */}
            {Debug && (
                <div className="text-[10px] text-muted bg-muted/20 p-1 rounded">
                    Debug: timestamp={selectedTimestamp}, timeidx={currentTimeIdx}, range={minTimeIdx}-{maxTimeIdx}
                </div>
            )}
            <div className="flex items-start">
                {/* Y-axis labels - only align with the graph area */}
                <div className="relative h-10 w-6 text-[10px] text-muted">
                    <span className="absolute top-[-5px] right-2">{maxCount}</span>
                    <span className="absolute top-[50%] right-2 transform -translate-y-1/2">
                        {Math.round(maxCount / 2)}
                    </span>
                    <span className="absolute bottom-[-7px] right-2">0</span>
                    {/* Tick marks that extend into the graph */}
                    <div className="absolute top-[0px] right-[-4px] w-[8px] h-px bg-secondary z-2"></div>
                    <div className="absolute top-[50%] right-[-4px] w-[8px] h-px bg-secondary transform -translate-y-1/2 z-2"></div>
                    <div className="absolute bottom-0 right-[-4px] w-[8px] h-px bg-secondary transform -translate-y-1/2 z-2"></div>
                </div>

                <div className="relative flex flex-col w-full">
                    {/* Area chart background */}
                    <div
                        className="relative h-10 bg-muted/10 cursor-pointer overflow-hidden border-l border-secondary"
                        onClick={handleGraphClick}
                    >
                        <svg className="absolute inset-0 w-full h-full" viewBox="0 0 100 40" preserveAspectRatio="none">
                            {/* Vertical grid lines */}
                            {GridPercentages.map((percent) => (
                                <line
                                    key={percent}
                                    x1={percent}
                                    y1="0"
                                    x2={percent}
                                    y2={GraphHeight}
                                    stroke="var(--color-secondary)"
                                    strokeOpacity="0.8"
                                    strokeWidth="0.5"
                                    vectorEffect="non-scaling-stroke"
                                />
                            ))}

                            {/* Horizontal midpoint line */}
                            <line
                                x1="0"
                                y1={GraphHeight / 2}
                                x2="100"
                                y2={GraphHeight / 2}
                                stroke="var(--color-secondary)"
                                strokeOpacity="0.8"
                                strokeWidth="0.5"
                                vectorEffect="non-scaling-stroke"
                            />

                            <path
                                d={createGoRoutineTimelinePath(
                                    sortedActiveCounts,
                                    minTimeIdx,
                                    plotTimeIdxRange,
                                    maxCount
                                )}
                                fill="var(--color-accent)"
                                fillOpacity="0.6"
                                stroke="var(--color-accent)"
                                strokeWidth="0.5"
                                vectorEffect="non-scaling-stroke"
                            />
                        </svg>

                        {/* Slider marker */}
                        <div
                            className="absolute w-[2px] bg-black pointer-events-none z-[1.5] rounded-lg"
                            style={{
                                ...(isAtLastPosition ? { right: "0px" } : { left: `${sliderMarkerPercent}%` }),
                                top: "-2px",
                                height: "calc(100% + 4px)",
                            }}
                        />
                    </div>

                    {/* Hidden range slider for accessibility and precise control */}
                    <input
                        type="range"
                        min={minTimeIdx}
                        max={maxTimeIdx}
                        step="1"
                        value={currentTimeIdx}
                        onChange={handleSliderChange}
                        className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                    />

                    {/* Time labels */}
                    <div className="relative">
                        {/* Start time */}
                        <span className="absolute left-0 text-[10px] text-muted">
                            {formatTime(sortedActiveCounts[0].tsts)}
                        </span>

                        {/* Intermediate labels with relative time */}
                        {GridPercentages.map((percent) => {
                            const relativeSeconds = Math.round((plotTimeIdxRange * percent) / 100);
                            return (
                                <span
                                    key={percent}
                                    className="absolute text-[10px] text-muted transform -translate-x-1/2"
                                    style={{ left: `${percent}%` }}
                                >
                                    +{relativeSeconds}s
                                </span>
                            );
                        })}

                        {/* End time */}
                        <span className="absolute right-0 text-[10px] text-muted">
                            {formatTime(sortedActiveCounts[sortedActiveCounts.length - 1].tsts)}
                        </span>
                    </div>
                </div>
            </div>
        </div>
    );
};
