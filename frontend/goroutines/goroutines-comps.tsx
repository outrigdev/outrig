// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Tooltip } from "@/elements/tooltip";
import { useAtomValue } from "jotai";
import React from "react";
import { GoRoutinesModel } from "./goroutines-model";

// Dropped Goroutines Indicator component
interface DroppedGoroutinesIndicatorProps {
    model: GoRoutinesModel;
}

export const DroppedGoroutinesIndicator = React.memo<DroppedGoroutinesIndicatorProps>(({ model }) => {
    const droppedCount = useAtomValue(model.droppedCount);

    if (droppedCount === 0) {
        return null;
    }

    const goroutineText = droppedCount === 1 ? "goroutine" : "goroutines";

    return (
        <div className="absolute bottom-0 right-0 bg-secondary/20 text-secondary rounded-tl-md px-2 py-1 text-xs z-10">
            <Tooltip content="Inactive goroutines are pruned after 10 minutes">
                <span className="font-normal cursor-default">
                    {droppedCount} {goroutineText} dropped
                </span>
            </Tooltip>
        </div>
    );
});
DroppedGoroutinesIndicator.displayName = "DroppedGoroutinesIndicator";
