import { useAtom, useAtomValue } from "jotai";
import React, { JSX, useRef } from "react";
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
    // Use a subtle red that works in light and dark mode.
    return <span className={srcStr === "stderr" ? "text-red-400 dark:text-red-300" : ""}>[{padded}]</span>;
}

const LogLineView: React.FC<LogLineViewProps> = ({ line }) => {
    if (line == null) {
        return null;
    }

    return (
        <div className="flex whitespace-nowrap">
            <div className="select-none pr-2 text-gray-400 w-12 text-right">{formatLineNumber(line.linenum, 4)}</div>
            <div>
                {formatTimestamp(line.ts, "HH:mm:ss.SSS")} {formatSource(line.source)} {line.msg}
            </div>
        </div>
    );
};

const LogViewer: React.FC<object> = () => {
    const model = useRef(new LogViewerModel()).current;
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredLogLines = useAtomValue(model.filteredLogLines);

    return (
        <div className="w-full h-full flex flex-col p-2">
            <input
                type="text"
                placeholder="Search logs..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full p-1 pl-2 mb-1 text-primary placeholder-muted border border-border rounded focus:outline-none focus:ring focus:ring-secondary"
            />
            <div className="w-full h-full overflow-auto flex-1">
                {/* Inner div - Forces min 1200px width and scrolls vertically */}
                <div className="w-full min-w-[1200px] h-full bg-white text-black font-mono text-xs leading-tight p-1 pt-2">
                    {filteredLogLines.map((line) => {
                        return <LogLineView key={line.linenum} line={line} />;
                    })}
                </div>
            </div>
        </div>
    );
};

export default LogViewer;
