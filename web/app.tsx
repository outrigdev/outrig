import { useAtom, useAtomValue } from "jotai";
import { Moon, Sun } from "lucide-react";
import { useEffect } from "react";
import { AppModel } from "./appmodel";
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

function MainTab() {
    const selectedTab = useAtomValue(AppModel.selectedTab);

    if (selectedTab === "logs") {
        return <LogViewer logIds={sampleLogIds} logLines={sampleLogLines} />;
    }

    return <div className="w-full h-full flex items-center justify-center text-secondary">Not Implemented</div>;
}

function Header() {
    const darkMode = useAtomValue(AppModel.darkMode);

    return (
        <header className="bg-panel text-primary p-4 border-b border-border">
            <div className="flex justify-between items-center">
                <div className="text-2xl font-bold">Outrig</div>
                <button
                    onClick={() => AppModel.setDarkMode(!darkMode)}
                    className="px-3 py-1 border border-border bg-button text-primary rounded text-sm flex items-center space-x-2 hover:bg-buttonhover transition-colors cursor-pointer"
                >
                    {darkMode ? <Moon size={16} /> : <Sun size={16} />}
                </button>
            </div>
        </header>
    );
}

function Tab({ name, displayName }: { name: string; displayName: string }) {
    const [selectedTab, setSelectedTab] = useAtom(AppModel.selectedTab);

    return (
        <button
            onClick={() => setSelectedTab(name)}
            data-selected={selectedTab === name || undefined}
            className="relative px-4 py-2 text-secondary text-sm data-[selected]:text-primary
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
            {displayName}
        </button>
    );
}

function App() {
    useEffect(() => {
        AppModel.applyTheme();
    }, []);

    return (
        <div className="h-screen w-screen flex flex-col bg-appbg">
            <Header />

            <nav className="bg-panel px-0.5 border-b-2 border-border flex">
                <Tab name="logs" displayName="Logs" />
                <Tab name="goroutines" displayName="GoRoutines" />
            </nav>

            {/* Main content */}
            <main className="flex-grow bg-panel overflow-auto w-full">
                <MainTab />
            </main>
        </div>
    );
}

export default App;
