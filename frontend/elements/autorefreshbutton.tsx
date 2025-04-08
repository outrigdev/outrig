// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { Atom, useAtomValue } from "jotai";
import { Clock, Timer } from "lucide-react";
import React, { useCallback } from "react";

interface AutoRefreshButtonProps {
    autoRefreshAtom: Atom<boolean>;
    onToggle: () => void;
    className?: string;
    size?: number;
}

export const AutoRefreshButton = React.memo<AutoRefreshButtonProps>(
    ({ autoRefreshAtom, onToggle, className, size = 16 }) => {
        const autoRefresh = useAtomValue(autoRefreshAtom);

        const handleToggle = useCallback(() => {
            onToggle();
        }, [onToggle]);

        return (
            <Tooltip
                content={autoRefresh ? "Auto-refresh On (Click to Disable)" : "Auto-refresh Off (Click to Enable)"}
            >
                <button
                    onClick={handleToggle}
                    className={cn(
                        "p-1 mr-1 rounded cursor-pointer transition-colors",
                        autoRefresh
                            ? "bg-primary/20 text-primary hover:bg-primary/30"
                            : "text-muted hover:bg-buttonhover hover:text-primary",
                        className
                    )}
                    aria-pressed={autoRefresh}
                >
                    {autoRefresh ? <Timer size={size} /> : <Clock size={size} />}
                </button>
            </Tooltip>
        );
    }
);
AutoRefreshButton.displayName = 'AutoRefreshButton';
