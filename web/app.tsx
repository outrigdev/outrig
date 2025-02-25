import { useState, useEffect } from "react";
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
  const [darkMode, setDarkMode] = useState(() => {
    return localStorage.getItem("theme") === "dark";
  });

  useEffect(() => {
    if (darkMode) {
      document.documentElement.dataset.theme = "dark";
      localStorage.setItem("theme", "dark");
    } else {
      document.documentElement.dataset.theme = "light";
      localStorage.setItem("theme", "light");
    }
  }, [darkMode]);

  return (
    <div className="h-screen w-screen flex flex-col bg-appbg">
      {/* Header */}
      <header className="bg-panel text-primary p-4 border-b border-border">
        <div className="flex justify-between items-center">
          <div className="text-2xl font-bold">Outrig</div>
          <button
            onClick={() => setDarkMode((prev) => !prev)}
            className="px-3 py-1 border border-border bg-button text-primary rounded text-sm flex items-center space-x-2 hover:bg-buttonhover transition-colors cursor-pointer"
          >
            {darkMode ? <Moon size={16} /> : <Sun size={16} />}
          </button>
        </div>
      </header>

      <nav className="bg-panel px-4 border-b-2 border-border flex">
        <button
          data-selected={true}
          className="relative px-4 py-2 text-primary text-sm 
             data-[selected]:after:content-[''] data-[selected]:after:absolute 
             data-[selected]:after:left-0 data-[selected]:after:bottom-[-2px] 
             data-[selected]:after:w-full data-[selected]:after:h-[2px] 
             data-[selected]:after:bg-primary
             hover:after:content-[''] hover:after:absolute 
             hover:after:left-0 hover:after:bottom-[-2px] 
             hover:after:w-full hover:after:h-[2px] 
             hover:after:bg-muted
             hover:text-primary transition-colors"
        >
          Logs
        </button>
        <button
          className="relative px-4 py-2 text-secondary text-sm 
             data-[selected]:after:content-[''] data-[selected]:after:absolute 
             data-[selected]:after:left-0 data-[selected]:after:bottom-[-2px] 
             data-[selected]:after:w-full data-[selected]:after:h-[2px] 
             data-[selected]:after:bg-primary
             hover:after:content-[''] hover:after:absolute 
             hover:after:left-0 hover:after:bottom-[-2px] 
             hover:after:w-full hover:after:h-[2px] 
             hover:after:bg-muted
             transition-colors"
        >
          GoRoutines
        </button>
      </nav>

      {/* Main content */}
      <main className="flex-grow bg-panel overflow-auto w-full">
        <LogViewer logIds={sampleLogIds} logLines={sampleLogLines} />
      </main>
    </div>
  );
}

export default App;
