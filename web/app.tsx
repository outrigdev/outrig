import { useAtom, useAtomValue } from "jotai";
import { Moon, Sun } from "lucide-react";
import { useEffect } from "react";
import { AppModel } from "./appmodel";
import LogViewer from "./logviewer/logviewer";

function MainTab() {
    const selectedTab = useAtomValue(AppModel.selectedTab);

    if (selectedTab === "logs") {
        return <LogViewer />;
    }

    return <div className="w-full h-full flex items-center justify-center text-secondary">Not Implemented</div>;
}

function AppLogo() {
    return (
        <div className="flex items-center space-x-2">
            <img src="/outriglogo.svg" alt="Outrig Logo" className="h-5" />
        </div>
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
                data-[selected]:after:left-0 data-[selected]:after:bottom-[-1px] 
                data-[selected]:after:w-full data-[selected]:after:h-[1px] 
                data-[selected]:after:bg-primary
                hover:after:content-[''] hover:after:absolute 
                hover:after:left-0 hover:after:bottom-[-1px] 
                hover:after:w-full hover:after:h-[1px] 
                hover:after:bg-muted
                hover:text-primary transition-colors cursor-pointer"
        >
            {displayName}
        </button>
    );
}

function App() {
    const darkMode = useAtomValue(AppModel.darkMode);

    useEffect(() => {
        AppModel.applyTheme();
    }, []);

    return (
        <div className="h-screen w-screen flex flex-col bg-panel">
            <nav className="bg-panel px-4 border-b border-border flex justify-between items-center">
                <div className="flex items-center">
                    <AppLogo />
                    <div className="ml-3 flex">
                        <Tab name="logs" displayName="Logs" />
                        <Tab name="goroutines" displayName="GoRoutines" />
                    </div>
                </div>
                <button
                    onClick={() => AppModel.setDarkMode(!darkMode)}
                    className="p-1.5 border border-border rounded-md text-primary hover:bg-buttonhover transition-colors cursor-pointer"
                >
                    {darkMode ? <Moon size={14} /> : <Sun size={14} />}
                </button>
            </nav>

            {/* Main content */}
            <main className="flex-grow overflow-auto w-full">
                <MainTab />
            </main>
        </div>
    );
}

export default App;
