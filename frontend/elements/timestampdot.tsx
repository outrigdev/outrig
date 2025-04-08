// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Tooltip } from "@/elements/tooltip";
import { formatRelativeTime } from "@/util/util";
import React, { useEffect, useRef, useState } from "react";

// Time in seconds for the pulse animation to complete
const PulseFadeTime = 3;

// TimestampDot component to show the last update time with animation
interface TimestampDotProps {
    timestamp: number;
}

export const TimestampDot: React.FC<TimestampDotProps> = ({ timestamp }) => {
    // Reference to the dot element for animation reset
    const dotRef = useRef<HTMLDivElement>(null);
    // Store the previous timestamp to detect changes
    const prevTsRef = useRef<number>(timestamp);
    // Use a key to force re-render when timestamp changes
    const [animationKey, setAnimationKey] = useState<number>(0);

    // Calculate how much time has passed since the update (in seconds)
    const timeSinceUpdate = Math.max(0, (Date.now() - timestamp) / 1000);

    // Determine if the timestamp is recent (less than 5 seconds old)
    const isRecent = timeSinceUpdate < PulseFadeTime;

    // Reset animation when timestamp changes
    useEffect(() => {
        if (prevTsRef.current !== timestamp) {
            // Update the key to force a re-render with fresh animation
            setAnimationKey((prev) => prev + 1);
            // Update the previous timestamp
            prevTsRef.current = timestamp;
        }
    }, [timestamp]);

    return (
        <Tooltip
            content={
                <div>
                    Updated: {new Date(timestamp).toLocaleTimeString()}
                    <span className="text-muted ml-1">({formatRelativeTime(timestamp)})</span>
                </div>
            }
        >
            <div
                key={animationKey}
                ref={dotRef}
                className="w-2.5 h-2.5 rounded-full"
                style={{
                    backgroundColor: isRecent ? "var(--watch-dot-active)" : "var(--watch-dot-inactive)",
                    animation: isRecent ? `pulse-fade ${PulseFadeTime}s linear forwards` : undefined,
                    opacity: isRecent ? 1 : 0.7,
                }}
            />
        </Tooltip>
    );
};
