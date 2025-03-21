import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { PrimitiveAtom, useAtomValue } from "jotai";
import { RefreshCw } from "lucide-react";
import React, { useState } from "react";

interface RefreshButtonProps {
    isRefreshingAtom: PrimitiveAtom<boolean>;
    onRefresh: () => void;
    tooltipContent?: string;
    className?: string;
    size?: number;
}

export const RefreshButton = React.memo<RefreshButtonProps>(
    ({ isRefreshingAtom, onRefresh, tooltipContent = "Refresh", className, size = 16 }) => {
        const isRefreshing = useAtomValue(isRefreshingAtom);
        const [isAnimating, setIsAnimating] = useState(false);

        const handleRefresh = () => {
            if (isRefreshing || isAnimating) return;

            // Start animation
            setIsAnimating(true);

            // Start refresh
            onRefresh();

            // End animation after 500ms
            setTimeout(() => {
                setIsAnimating(false);
            }, 500);
        };

        return (
            <Tooltip content={tooltipContent}>
                <button
                    onClick={handleRefresh}
                    className={cn(
                        "p-1 mr-1 rounded hover:bg-buttonhover text-muted hover:text-primary cursor-pointer",
                        isAnimating && "refresh-spin",
                        className
                    )}
                    disabled={isRefreshing || isAnimating}
                >
                    <RefreshCw size={size} />
                </button>
            </Tooltip>
        );
    }
);
RefreshButton.displayName = 'RefreshButton';
