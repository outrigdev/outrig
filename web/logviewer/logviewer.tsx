import { useAtom, useAtomValue } from "jotai";
import { Filter } from "lucide-react";
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
    return <span className={srcStr === "stderr" ? "text-error" : "text-muted"}>[{padded}]</span>;
}

const LogLineView: React.FC<LogLineViewProps> = ({ line }) => {
    if (line == null) {
        return null;
    }

    return (
        <div className="flex whitespace-nowrap hover:bg-buttonhover py-0.5">
            <div className="select-none pr-2 text-muted w-12 text-right">{formatLineNumber(line.linenum, 4)}</div>
            <div>
                <span className="text-secondary">{formatTimestamp(line.ts, "HH:mm:ss.SSS")}</span>{" "}
                {formatSource(line.source)} <span className="text-primary">{line.msg}</span>
            </div>
        </div>
    );
};

const LogViewer: React.FC<object> = () => {
    const model = useRef(new LogViewerModel()).current;
    const [search, setSearch] = useAtom(model.searchTerm);
    const filteredLogLines = useAtomValue(model.filteredLogLines);

    return (
        <div className="w-full h-full flex flex-col">
            {/* Search container with icon */}
            <div className="py-1">
                <div className="relative">
                    <div className="absolute inset-y-0 left-0 flex items-center pl-2 pointer-events-none text-muted">
                        <Filter size={16} fill="currentColor" stroke="currentColor" strokeWidth={1} />
                    </div>
                    <input
                        type="text"
                        placeholder="filter..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-full bg-transparent text-primary placeholder:text-muted text-sm py-1 pl-8 pr-2 
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

export default LogViewer;
