import { AppModel } from "@/appmodel";
import { AnsiLine } from "@/elements/ansiline";
import { SettingsModel } from "@/settings/settings-model";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import React, { useCallback } from "react";
import { LogViewerModel } from "./logviewer-model";

function formatMarkedLineNumber(num: number, isMarked: boolean, width: number, onClick: () => void): React.ReactNode {
    const paddedNum = String(num).padStart(isMarked ? width : width + 1, " ");

    if (isMarked) {
        return (
            <div
                className="text-right flex-shrink-0 cursor-pointer text-accent hover:text-primary whitespace-pre"
                onClick={onClick}
            >
                <span className="text-accent">•</span>
                {paddedNum}
            </div>
        );
    }

    return (
        <div className="text-right flex-shrink-0 cursor-pointer hover:text-primary whitespace-pre" onClick={onClick}>
            {paddedNum}
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
    const padded = srcStr.padStart(6, " ");
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

// LogLineComponent for rendering individual log lines in LogVList
interface LogLineComponentProps {
    line: LogLine;
    model?: LogViewerModel;
}

export const LogLineComponent = React.memo<LogLineComponentProps>(({ line, model }) => {
    useAtomValue(model.markedLinesVersion);
    const lineNumWidth = useAtomValue(model.lineNumberWidth);
    const showSource = useAtomValue(SettingsModel.logsShowSource);
    const showMilliseconds = useAtomValue(SettingsModel.logsShowMilliseconds);
    const timeFormat = useAtomValue(SettingsModel.logsTimeFormat);
    const appRunStartTime = useAtomValue(AppModel.appRunStartTimeAtom);

    const handleLineNumberClick = useCallback(() => {
        model.toggleLineMarked(line.linenum);
    }, [model, line.linenum]);

    const isMarked = model.isLineMarked(line.linenum);

    return (
        <div
            data-linenum={line.linenum}
            className={cn("flex text-muted select-none", isMarked ? "bg-accentbg/20" : "hover:bg-buttonhover")}
        >
            {formatMarkedLineNumber(line.linenum, isMarked, lineNumWidth, handleLineNumberClick)}
            <div className="text-secondary flex-shrink-0 pl-2">
                {formatTimestamp(line.ts, showMilliseconds, timeFormat, appRunStartTime)}
            </div>
            {showSource && <div className="pl-2">{formatSource(line.source)}</div>}
            <AnsiLine
                className="flex-1 min-w-0 pl-2 select-text text-primary break-all overflow-hidden whitespace-pre"
                line={line.msg}
            />
        </div>
    );
});
LogLineComponent.displayName = "LogLineComponent";
