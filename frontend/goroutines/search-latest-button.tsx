// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import { SkipForward } from "lucide-react";
import React from "react";
import { GoRoutinesModel } from "./goroutines-model";

interface SearchLatestButtonProps {
    model: GoRoutinesModel;
}

export const SearchLatestButton: React.FC<SearchLatestButtonProps> = ({ model }) => {
    const searchLatestMode = useAtomValue(model.searchLatestMode);
    const search = useAtomValue(model.searchTerm);

    const handleSearchLatest = () => {
        model.enableSearchLatest();
        model.searchGoroutines(search);
    };

    return (
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
    );
};