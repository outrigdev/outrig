import { Atom, atom, PrimitiveAtom } from "jotai";

// Sample data
const sampleLogLines: LogLine[] = [
    {
        linenum: 1,
        ts: Date.now(),
        msg: "This is the first log line",
        source: "/dev/stdout",
    },
    {
        linenum: 2,
        ts: Date.now(),
        msg: "Another log entry here",
        source: "/dev/stderr",
    },
    {
        linenum: 3,
        ts: Date.now(),
        msg: "Yet another log line",
        source: "/dev/stdout",
    },
];

class LogViewerModel {
    searchTerm: PrimitiveAtom<string> = atom("");
    filteredLogLines: Atom<LogLine[]> = atom((get) => {
        const search = get(this.searchTerm);
        return sampleLogLines.filter((log) => log.msg.toLowerCase().includes(search.toLowerCase()));
    });

    getLogLines(): LogLine[] {
        return sampleLogLines;
    }
}

export { LogViewerModel };
