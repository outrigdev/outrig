// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { PrimitiveAtom, useAtom, useAtomValue } from "jotai";
import { Layers, Layers2, Search } from "lucide-react";
import React, { useCallback } from "react";
import { GoRoutinesModel } from "./goroutines-model";

interface StacktraceModeToggleProps {
    modeAtom: PrimitiveAtom<string>;
    model: GoRoutinesModel;
}

export const StacktraceModeToggle: React.FC<StacktraceModeToggleProps> = ({ modeAtom, model }) => {
    const [mode, setMode] = useAtom(modeAtom);
    const searchTerm = useAtomValue(model.searchTerm);
    const isSearchActive = searchTerm && searchTerm.trim() !== "";

    const handleToggleMode = useCallback(() => {
        if (isSearchActive) return;

        if (mode === "raw") {
            setMode("simplified");
        } else if (mode === "simplified") {
            setMode("simplified:files");
        } else {
            setMode("raw");
        }
    }, [mode, setMode, isSearchActive]);

    const tooltipContent = useCallback(() => {
        if (isSearchActive) {
            return "Raw Stacktrace Mode Locked (to reveal search matches)";
        }

        switch (mode) {
            case "raw":
                return "Raw Stacktrace Mode (Click to Toggle)";
            case "simplified":
                return "Simplified Stacktrace Mode (Click to Toggle)";
            case "simplified:files":
                return "Simplified Stacktrace with Files Mode (Click to Toggle)";
            default:
                return "Toggle Stacktrace Mode";
        }
    }, [mode, isSearchActive]);

    const renderIcon = useCallback(() => {
        switch (mode) {
            case "simplified":
                return <Layers2 size={16} />;
            case "simplified:files":
                return <Layers size={16} />;
            case "raw":
            default:
                return <Layers size={16} />;
        }
    }, [mode]);

    return (
        <Tooltip content={tooltipContent()}>
            <button
                onClick={handleToggleMode}
                className={cn(
                    "p-1 mr-1 rounded transition-colors relative",
                    isSearchActive ? "cursor-default" : "cursor-pointer",
                    mode !== "raw"
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                )}
                aria-pressed={mode !== "raw" ? "true" : "false"}
            >
                {renderIcon()}

                {isSearchActive && (
                    <div className="absolute -top-1 -right-1 bg-accent rounded-full p-0.5">
                        <Search size={10} className="text-white" />
                    </div>
                )}
            </button>
        </Tooltip>
    );
};
