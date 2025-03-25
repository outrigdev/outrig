import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import React, { useCallback } from "react";
import { LogViewerModel } from "./logviewer-model";

function formatMarkedLineNumber(num: number, isMarked: boolean, width = 4): React.ReactNode {
    const paddedNum = String(num).padStart(width, " ");
    if (isMarked) {
        return (
            <span className="text-primary">
                <span className="text-accent">•</span> {paddedNum}
            </span>
        );
    }
    return <> {paddedNum}</>;
}

function formatTimestamp(ts: number, format: string = "HH:mm:ss.SSS") {
    const date = new Date(ts);
    const hh = date.getHours().toString().padStart(2, "0");
    const mm = date.getMinutes().toString().padStart(2, "0");
    const ss = date.getSeconds().toString().padStart(2, "0");
    const sss = date.getMilliseconds().toString().padStart(3, "0");
    return `${hh}:${mm}:${ss}.${sss}`;
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
    // Subscribe to the version atom to trigger re-renders when marked lines change
    useAtomValue(model.markedLinesVersion);

    const handleLineNumberClick = useCallback(() => {
        model.toggleLineMarked(line.linenum);
    }, [model, line.linenum]);

    const isMarked = model.isLineMarked(line.linenum);

    return (
        <div
            data-linenum={line.linenum}
            className={cn("flex text-muted select-none", isMarked ? "bg-accentbg/20" : "hover:bg-buttonhover")}
        >
            <div
                className={cn(
                    "w-12 text-right flex-shrink-0 cursor-pointer",
                    isMarked ? "text-accent" : "hover:text-primary"
                )}
                onClick={handleLineNumberClick}
            >
                {formatMarkedLineNumber(line.linenum, isMarked, 4)}
            </div>
            <div className="text-secondary flex-shrink-0 pl-2">{formatTimestamp(line.ts, "HH:mm:ss.SSS")}</div>
            <div className="pl-2">{formatSource(line.source)}</div>
            <div className="flex-1 min-w-0 pl-2 select-text">
                <span className="text-primary break-all overflow-hidden whitespace-pre">{line.msg}</span>
            </div>
        </div>
    );
});
LogLineComponent.displayName = "LogLineComponent";
