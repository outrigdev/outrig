// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AnsiLine } from "@/elements/ansiline";
import { LogSettings } from "@/settings/settings-model";
import { cn } from "@/util/util";
import EmojiJS from "emoji-js";
import { getDefaultStore, useAtomValue } from "jotai";
import React, { useCallback, useMemo } from "react";
import { LogViewerModel } from "./logviewer-model";

// Initialize emoji parser
const emoji = new EmojiJS();
emoji.replace_mode = "unified";
emoji.allow_native = true;

// Global variable for emoji replacement setting
let emojiReplacementMode: "never" | "outrig" | "always" = "outrig";

// Initialize from settings
import { SettingsModel } from "@/settings/settings-model";

// Map of sources that should have emoji replacement when in "outrig" mode
const outrigEmojiReplacementSources: Record<string, boolean> = {
    outrig: true,
};

// Update the global variable when settings change
const updateEmojiReplacementMode = () => {
    const store = getDefaultStore();
    emojiReplacementMode = store.get(SettingsModel.logsEmojiReplacement);
};

// Initialize the emoji replacement mode
updateEmojiReplacementMode();

// Subscribe to settings changes
getDefaultStore().sub(SettingsModel.logsEmojiReplacement, () => {
    updateEmojiReplacementMode();
});

// Interface for combined log line settings
interface LogLineSettings {
    lineNumWidth: number;
    logSettings: LogSettings;
    appRunStartTime: number | null;
}

function formatMarkedLineNumber(
    num: number,
    isMarked: boolean,
    width: number,
    onClick: () => void,
    showLineNumbers: boolean
): React.ReactNode {
    // When line numbers are hidden, width is always 1
    const effectiveWidth = showLineNumbers ? width : 0;

    // For hidden line numbers, just use a space or bullet
    const content = showLineNumbers ? String(num) : "";
    const paddedContent = content.padStart(isMarked ? effectiveWidth : effectiveWidth + 1, " ");

    if (isMarked) {
        return (
            <>
                {/* prettier-ignore */}
                <div
					className="text-right flex-shrink-0 cursor-pointer text-accent hover:text-primary whitespace-pre"
					onClick={onClick}
				>
					<span className="text-accent">•</span>{paddedContent}
				</div>
            </>
        );
    }

    return (
        <div className="text-right flex-shrink-0 cursor-pointer hover:text-primary whitespace-pre" onClick={onClick}>
            {paddedContent}
        </div>
    );
}

function formatTimestamp(ts: number, showMilliseconds: boolean, timeFormat: string, referenceTime?: number) {
    const date = new Date(ts);

    if (timeFormat === "relative" && referenceTime != null) {
        // Calculate time difference in milliseconds
        const diff = ts - referenceTime;

        // Format as +/-MM:SS.mmm
        const isNegative = diff < 0;
        const absDiff = Math.abs(diff);
        const minutes = Math.floor(absDiff / 60000)
            .toString()
            .padStart(2, "0");
        const seconds = Math.floor((absDiff % 60000) / 1000)
            .toString()
            .padStart(2, "0");
        const milliseconds = (absDiff % 1000).toString().padStart(3, "0");

        if (showMilliseconds) {
            return `${isNegative ? "-" : "+"}${minutes}:${seconds}.${milliseconds}`;
        } else {
            return `${isNegative ? "-" : "+"}${minutes}:${seconds}`;
        }
    } else {
        // Absolute time format
        const hh = date.getHours().toString().padStart(2, "0");
        const mm = date.getMinutes().toString().padStart(2, "0");
        const ss = date.getSeconds().toString().padStart(2, "0");
        const sss = date.getMilliseconds().toString().padStart(3, "0");

        if (showMilliseconds) {
            return `${hh}:${mm}:${ss}.${sss}`;
        } else {
            return `${hh}:${mm}:${ss}`;
        }
    }
}

function formatSource(source: string): React.ReactNode {
    let srcStr = source || "";
    if (srcStr.startsWith("/dev/")) {
        srcStr = srcStr.slice(5);
    }

    // Store original source for tooltip
    const originalSrc = srcStr;
    const isTruncated = srcStr.length > 6;

    // Limit source to max 6 chars
    if (isTruncated) {
        srcStr = srcStr.substring(0, 6);
    }

    const padded = srcStr.padStart(6, " ");
    let className = "text-muted";
    if (srcStr == "stdout") {
        className = "text-muted";
    } else if (srcStr === "stderr") {
        className = "text-error";
    } else if (srcStr === "outrig") {
        className = "text-accent";
    } else {
        className = "text-ansi-brightmagenta";
    }

    return (
        <span className={className} title={isTruncated ? "Source: " + originalSrc : undefined}>
            [{padded}]
        </span>
    );
}

// Process message text with emoji replacement if needed
function processMessageText(message: string, source: string): string {
    // Check emoji replacement mode
    if (emojiReplacementMode === "always") {
        // Always replace emojis regardless of source
        return emoji.replace_colons(message);
    } else if (emojiReplacementMode === "outrig" && outrigEmojiReplacementSources[source]) {
        // Replace emojis only for specified sources in "outrig" mode
        return emoji.replace_colons(message);
    }

    // Return original message if no replacement needed
    return message;
}

// LogLineComponent for rendering individual log lines in LogVList
interface LogLineComponentProps {
    line: LogLine;
    model?: LogViewerModel;
    lineSettings: LogLineSettings;
    pageNum: number;
    lineIndex: number;
    onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void;
}

export const LogLineComponent = React.memo<LogLineComponentProps>(({ line, model, lineSettings, pageNum, lineIndex, onContextMenu }) => {
    useAtomValue(model.markedLinesVersion);
    const { lineNumWidth, logSettings, appRunStartTime } = lineSettings;

    const handleLineNumberClick = useCallback(() => {
        model.toggleLineMarked(line.linenum);
    }, [model, line.linenum]);

    const handleContextMenu = useCallback((e: React.MouseEvent) => {
        if (onContextMenu) {
            e.preventDefault();
            onContextMenu(e, pageNum, lineIndex);
        }
    }, [onContextMenu, pageNum, lineIndex]);

    const isMarked = model.isLineMarked(line.linenum);

    // Process message with emoji replacement if needed
    const processedMessage = useMemo(() => {
        return processMessageText(line.msg, line.source);
    }, [line.msg, line.source]);

    return (
        <div
            data-linenum={line.linenum}
            data-linepage={pageNum}
            data-lineindex={lineIndex}
            onContextMenu={handleContextMenu}
            className={cn(
                "flex text-muted select-none pl-1 pr-2",
                isMarked ? "bg-accentbg/20" : "hover:bg-buttonhover"
            )}
        >
            {formatMarkedLineNumber(
                line.linenum,
                isMarked,
                lineNumWidth,
                handleLineNumberClick,
                logSettings.showLineNumbers
            )}
            {logSettings.showTimestamp && (
                <div className="text-secondary flex-shrink-0 pl-2">
                    {formatTimestamp(line.ts, logSettings.showMilliseconds, logSettings.timeFormat, appRunStartTime)}
                </div>
            )}
            {logSettings.showSource && <div className="pl-2">{formatSource(line.source)}</div>}
            <AnsiLine
                className="flex-1 min-w-0 pl-2 select-text text-primary break-all overflow-hidden whitespace-pre"
                line={processedMessage}
            />
        </div>
    );
});
LogLineComponent.displayName = "LogLineComponent";
