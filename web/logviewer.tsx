import React, { JSX, useState } from "react";

interface LogLineViewProps {
    logNum: number;
    logLines: Map<number, LogLine>;
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

const LogLineView: React.FC<LogLineViewProps> = ({ logNum, logLines }) => {
    const log = logLines.get(logNum);
    if (!log) return null;

    return (
        <div className="flex whitespace-nowrap">
            <div className="select-none pr-2 text-gray-400 w-12 text-right">{formatLineNumber(log.linenum, 4)}</div>
            <div>
                {formatTimestamp(log.ts, "HH:mm:ss.SSS")} {formatSource(log.source)} {log.msg}
            </div>
        </div>
    );
};

interface LogViewerProps {
    logIds: number[];
    logLines: Map<number, LogLine>;
}

const LogViewer: React.FC<LogViewerProps> = ({ logIds, logLines }) => {
    const [search, setSearch] = useState("");

    const filteredLogIds = logIds.filter((id) => {
        const log = logLines.get(id);
        if (!log) return false;
        return log.msg.toLowerCase().includes(search.toLowerCase());
    });

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
                    {filteredLogIds.map((id) => {
                        return <LogLineView key={id} logNum={id} logLines={logLines} />;
                    })}
                </div>
            </div>
        </div>
    );
};

export default LogViewer;
