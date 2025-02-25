import React, { useState } from "react";

type LogLine = {
  linenum: number;
  ts: number; // unix time in ms
  msg: string;
  source: string;
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
        className="w-full p-1 mb-2 border border-gray-300 dark:border-gray-600 rounded focus:outline-none focus:ring"
      />
      <div className="w-full h-full overflow-auto flex-1">
        {/* Inner div - Forces min 1200px width and scrolls vertically */}
        <div className="w-full min-w-[1200px] h-full bg-white dark:bg-black text-black dark:text-white font-mono text-xs leading-tight">
          {filteredLogIds.map((id) => {
            const log = logLines.get(id);
            if (!log) return null;
            return (
              <div key={id} className="whitespace-nowrap">
                {new Date(log.ts).toLocaleTimeString()} {log.source} {log.msg}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default LogViewer;
