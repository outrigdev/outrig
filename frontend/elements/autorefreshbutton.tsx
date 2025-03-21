import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { PrimitiveAtom, useAtom } from "jotai";
import { Clock, Timer } from "lucide-react";
import React, { useCallback } from "react";

interface AutoRefreshButtonProps {
    autoRefreshAtom: PrimitiveAtom<boolean>;
    onToggle?: () => void;
    className?: string;
    size?: number;
}

export const AutoRefreshButton = React.memo<AutoRefreshButtonProps>(
    ({ autoRefreshAtom, onToggle, className, size = 16 }) => {
        const [autoRefresh, setAutoRefresh] = useAtom(autoRefreshAtom);

        const handleToggle = useCallback(() => {
            setAutoRefresh(!autoRefresh);
            if (onToggle) {
                onToggle();
            }
        }, [autoRefresh, setAutoRefresh, onToggle]);

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
