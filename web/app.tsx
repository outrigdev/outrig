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

function Header({ full }: { full: boolean }) {
    if (full) {
        return <HeaderFull />;
    } else {
        return <HeaderSmall />;
    }
}

function HeaderFull() {
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

function HeaderSmall() {
    const darkMode = useAtomValue(AppModel.darkMode);

    return (
        <header className="bg-panel text-primary py-1 px-3 border-b border-border">
            <div className="flex justify-between items-center">
                <div className="flex items-center space-x-2">
                    <img
                        src="/outriglogo.svg"
                        alt="Outrig Logo"
                        className="h-6"
                        style={{ height: "1.125rem" }} // Match text-lg (1.125rem)
                    />
                    <div className="text-lg font-semibold">Outrig</div>
                </div>
                <button
                    onClick={() => AppModel.setDarkMode(!darkMode)}
                    className="px-2 py-0.5 border border-border bg-button text-primary rounded text-xs flex items-center space-x-1 hover:bg-buttonhover transition-colors cursor-pointer"
                >
                    {darkMode ? <Moon size={14} /> : <Sun size={14} />}
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
hover:text-primary transition-colors cursor-pointer"
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
            <Header full={false} />

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
