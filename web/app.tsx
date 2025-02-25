import { useState } from "react";
import { Sun, Moon } from "lucide-react";
import LogViewer from "./logviewer";

// Sample data
const sampleLogLines: Map<number, LogLine> = new Map([
  [
    1,
    {
      linenum: 1,
      ts: Date.now(),
      msg: "This is the first log line",
      source: "/dev/stdout",
    },
  ],
  [
    2,
    {
      linenum: 2,
      ts: Date.now(),
      msg: "Another log entry here",
      source: "/dev/stderr",
    },
  ],
  [
    3,
    {
      linenum: 3,
      ts: Date.now(),
      msg: "Yet another log line",
      source: "/dev/stdout",
    },
  ],
]);

const sampleLogIds = Array.from(sampleLogLines.keys());

function App() {
  const [darkMode, setDarkMode] = useState(false);

  return (
    <div
      className={`${darkMode ? "dark" : ""} h-screen w-screen flex flex-col`}
    >
      {/* Header */}
      <header className="bg-gray-100 dark:bg-gray-800 p-4">
        <div className="flex justify-between items-center">
          <div className="text-2xl font-bold">Outrig</div>
          <button
            onClick={() => setDarkMode((prev) => !prev)}
            className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm flex items-center space-x-2"
          >
            {darkMode ? <Sun size={16} /> : <Moon size={16} />}
            <span>{darkMode ? "Light Mode" : "Dark Mode"}</span>
          </button>
        </div>
      </header>

      {/* Tabs row */}
      <nav className="bg-gray-200 dark:bg-gray-700 px-4 py-2">
        <button className="px-4 py-1 bg-white dark:bg-gray-600 text-black dark:text-white rounded shadow-sm">
          Logs
        </button>
      </nav>

      {/* Main content */}
      <main className="flex-grow bg-gray-50 dark:bg-gray-900 overflow-auto w-full">
        <LogViewer logIds={sampleLogIds} logLines={sampleLogLines} />
      </main>
    </div>
  );
}

export default App;
