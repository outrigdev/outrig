import { useAtom, useAtomValue } from "jotai";
import { Filter } from "lucide-react";
import React, { JSX, useEffect, useRef } from "react";
import { AppModel } from "../appmodel";
import { LogViewerModel } from "./logviewer-model";

interface LogLineViewProps {
    line: LogLine;
}

function formatLineNumber(num: number, width = 4) {
    return String(num).padStart(width, " ");
}

function formatTimestamp(ts: number, format: string = "HH:mm:ss.SSS") {
    const date = new Date(ts);
    const hh = date.getHours().toString().padStart(2, "0");
    const mm = date.getMinutes().toString().padStart(2, "0");
    const ss = date.getSeconds().toString().padStart(2, "0");
    const sss = date.getMilliseconds().toString().padStart(3, "0");
    return `${hh}:${mm}:${ss}.${sss}`;
}

function formatSource(source: string): JSX.Element {
    let srcStr = source || "";
    if (srcStr.startsWith("/dev/")) {
        srcStr = srcStr.slice(5);
    }
    const padded = srcStr.padEnd(6, " ");
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

const LogLineView: React.FC<LogLineViewProps> = ({ line }) => {
    if (line == null) {
        return null;
    }

    return (
        <div className="flex whitespace-nowrap hover:bg-buttonhover">
            <div className="select-none pr-2 text-muted w-12 text-right">{formatLineNumber(line.linenum, 4)}</div>
            <div>
                <span className="text-secondary">{formatTimestamp(line.ts, "HH:mm:ss.SSS")}</span>{" "}
                {formatSource(line.source)} <span className="text-primary">{line.msg}</span>
            </div>
        </div>
    );
};
// LogViewer component with better alignment
export const LogViewer: React.FC<object> = () => {
    const model = useRef(new LogViewerModel()).current;
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredLogLines = useAtomValue(model.filteredLogLines);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);
    const appRuns = useAtomValue(AppModel.appRuns);
    const searchRef = useRef<HTMLInputElement>(null);

    // Find the selected app run
    const selectedAppRun = appRuns.find(run => run.apprunid === selectedAppRunId);

    useEffect(() => {
        // on window focus, focus the search input
        const onFocus = () => {
            searchRef.current?.focus();
        };
        window.addEventListener("focus", onFocus);
        return () => {
            window.removeEventListener("focus", onFocus);
        };
    }, []);

    return (
        <div className="w-full h-full flex flex-col">
            <div className="py-2 px-4 border-b border-border">
                <div className="flex justify-between items-center">
                    <h2 className="text-lg font-semibold text-primary">
                        {selectedAppRun ? `Logs: ${selectedAppRun.appname}` : 'Logs'}
                    </h2>
                    {selectedAppRun && (
                        <div className="text-xs text-muted">
                            ID: {selectedAppRun.apprunid}
                        </div>
                    )}
                </div>
            </div>

            <div className="py-1 px-1 border-b border-border">
                <div className="flex items-center">
                    {/* Line number space - 6 characters wide with right-aligned filter icon */}
                    <div className="select-none pr-2 text-muted w-12 text-right font-mono flex justify-end items-center">
                        <Filter
                            size={16}
                            className="text-muted"
                            fill="currentColor"
                            stroke="currentColor"
                            strokeWidth={1}
                        />
                    </div>

                    {/* Filter input */}
                    <input
                        ref={searchRef}
                        type="text"
                        placeholder="filter..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                   border-none ring-0 outline-none focus:outline-none focus:ring-0"
                    />
                </div>
            </div>

            {/* Subtle divider */}
            <div className="h-px bg-border"></div>

            {/* Log content */}
            <div className="w-full h-full overflow-auto flex-1 px-1 pt-2">
                <div className="w-full min-w-[1200px] h-full font-mono text-xs leading-tight">
                    {filteredLogLines.map((line) => {
                        return <LogLineView key={line.linenum} line={line} />;
                    })}
                </div>
            </div>
        </div>
    );
};
