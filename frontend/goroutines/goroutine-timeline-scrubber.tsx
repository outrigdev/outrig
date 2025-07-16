// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import React from "react";
import { GoRoutinesModel } from "./goroutines-model";

interface GoRoutineTimelineScrubberProps {
    model: GoRoutinesModel;
}

export const GoRoutineTimelineScrubber: React.FC<GoRoutineTimelineScrubberProps> = ({ model }) => {
    const activeCounts = useAtomValue(model.activeCounts);
    const selectedTimestamp = useAtomValue(model.selectedTimestamp);
    const searchLatestMode = useAtomValue(model.searchLatestMode);
    const fullTimeSpan = useAtomValue(model.fullTimeSpan);
    const appRunInfoAtom = AppModel.getAppRunInfoAtom(model.appRunId);
    const appRunInfo = useAtomValue(appRunInfoAtom);
    const isAppRunning = useAtomValue(AppModel.selectedAppRunIsRunningAtom);

    if (!appRunInfo || !fullTimeSpan || activeCounts.length === 0) {
        return null;
    }

    // Sort active counts by timeidx to ensure proper ordering
    const sortedActiveCounts = [...activeCounts].sort((a, b) => a.timeidx - b.timeidx);

    if (sortedActiveCounts.length === 0) {
        return null;
    }

    // Get the time index range
    const minTimeIdx = sortedActiveCounts[0].timeidx;
    const maxTimeIdx = sortedActiveCounts[sortedActiveCounts.length - 1].timeidx;
    const timeIdxRange = maxTimeIdx - minTimeIdx;

    if (timeIdxRange === 0) {
        return null;
    }

    // Find max count for scaling
    const maxCount = Math.max(...sortedActiveCounts.map((c) => c.count));
    if (maxCount === 0) {
        return null;
    }

    // Helper function to find the closest active count entry for a given timeidx
    const findActiveCountByTimeIdx = (timeIdx: number): GoRoutineActiveCount | null => {
        return sortedActiveCounts.find((count) => count.timeidx === timeIdx) || null;
    };

    // Helper function to convert timeidx to timestamp
    const timeIdxToTimestamp = (timeIdx: number): number => {
        const activeCount = findActiveCountByTimeIdx(timeIdx);
        return activeCount ? activeCount.tsts : 0;
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
    const currentTimeIdx =
        searchLatestMode || selectedTimestamp === 0 ? maxTimeIdx : timestampToTimeIdx(selectedTimestamp);

    // Calculate container width - same logic as goroutines table
    let containerWidthStyle: string;
    if (!isAppRunning) {
        containerWidthStyle = "100%";
    } else {
        // For running app: pad timeline to 15s boundary
        const startTime = fullTimeSpan.start;
        const endTime = fullTimeSpan.end;
        const timelineDurationSeconds = (endTime - startTime) / 1000;
        const paddedTimelineSeconds = Math.max(Math.ceil(timelineDurationSeconds / 15) * 15, 15);
        const widthPercent = (timelineDurationSeconds / paddedTimelineSeconds) * 100;
        containerWidthStyle = `${widthPercent}%`;
    }

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

        // Convert click position to timeidx
        const clickedTimeIdx = Math.round(minTimeIdx + clickPercent * timeIdxRange);
        const timestamp = timeIdxToTimestamp(clickedTimeIdx);

        if (timestamp > 0) {
            model.setSelectedTimestampAndSearch(timestamp);
        }
    };

    // Create SVG path for area chart using timeidx as x-axis
    const createAreaPath = (): string => {
        if (sortedActiveCounts.length === 0) return "";

        const graphHeight = 40; // Height of the graph area

        let path = `M 0 ${graphHeight}`; // Start at bottom left

        sortedActiveCounts.forEach((count, index) => {
            const x = ((count.timeidx - minTimeIdx) / timeIdxRange) * 100;
            const y = graphHeight - (count.count / maxCount) * graphHeight;

            if (index === 0) {
                path += ` L ${x} ${y}`;
            } else {
                path += ` L ${x} ${y}`;
            }
        });

        // Close the path at bottom right
        path += ` L 100 ${graphHeight} Z`;

        return path;
    };

    // Calculate slider position marker
    let sliderMarkerPercent: number | null = null;
    if (searchLatestMode) {
        sliderMarkerPercent = 100;
    } else if (selectedTimestamp > 0) {
        const timeIdx = timestampToTimeIdx(selectedTimestamp);
        sliderMarkerPercent = ((timeIdx - minTimeIdx) / timeIdxRange) * 100;
    }

    return (
        <div className="flex flex-col gap-2">
            <div className="text-xs text-muted font-medium">Active Goroutines</div>
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

                <div className="relative flex flex-col" style={{ width: containerWidthStyle }}>
                    {/* Area chart background */}
                    <div
                        className="relative h-10 bg-muted/10 rounded-tr rounded-br cursor-pointer overflow-hidden border-l border-secondary"
                        onClick={handleGraphClick}
                    >
                        <svg className="absolute inset-0 w-full h-full" viewBox="0 0 100 40" preserveAspectRatio="none">
                            {/* Vertical grid lines */}
                            {[25, 50, 75].map((percent) => (
                                <line
                                    key={percent}
                                    x1={percent}
                                    y1="0"
                                    x2={percent}
                                    y2="40"
                                    stroke="var(--color-secondary)"
                                    strokeOpacity="0.8"
                                    strokeWidth="0.5"
                                    vectorEffect="non-scaling-stroke"
                                />
                            ))}
                            
                            {/* Horizontal midpoint line */}
                            <line
                                x1="0"
                                y1="20"
                                x2="100"
                                y2="20"
                                stroke="var(--color-secondary)"
                                strokeOpacity="0.8"
                                strokeWidth="0.5"
                                vectorEffect="non-scaling-stroke"
                            />
                            
                            <path
                                d={createAreaPath()}
                                fill="var(--color-accent)"
                                fillOpacity="0.6"
                                stroke="var(--color-accent)"
                                strokeWidth="0.5"
                                vectorEffect="non-scaling-stroke"
                            />
                        </svg>

                        {/* Slider marker */}
                        {sliderMarkerPercent !== null && (
                            <div
                                className="absolute w-[2px] bg-black pointer-events-none z-[1.5] rounded-lg"
                                style={{
                                    left: `${sliderMarkerPercent}%`,
                                    top: "-2px",
                                    height: "calc(100% + 4px)",
                                }}
                            />
                        )}
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
                            {new Date(sortedActiveCounts[0].tsts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                        </span>
                        
                        {/* Intermediate labels with relative time */}
                        {[25, 50, 75].map((percent) => {
                            const startTime = sortedActiveCounts[0].tsts;
                            const endTime = sortedActiveCounts[sortedActiveCounts.length - 1].tsts;
                            const timeAtPercent = startTime + (endTime - startTime) * (percent / 100);
                            const relativeSeconds = Math.round((timeAtPercent - startTime) / 1000);
                            
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
                            {new Date(sortedActiveCounts[sortedActiveCounts.length - 1].tsts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                        </span>
                    </div>
                </div>
            </div>
        </div>
    );
};
