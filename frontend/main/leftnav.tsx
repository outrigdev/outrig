import { useAtom, useAtomValue } from "jotai";
import { X, Home, Settings, Github, BookOpen, MessageSquare, Youtube, Twitter } from "lucide-react";
import { AppModel } from "../appmodel";
import { cn } from "../util/util";
import { useMemo } from "react";

export const LeftNav: React.FC = () => {
    const [isOpen, setIsOpen] = useAtom(AppModel.leftNavOpen);
    const unsortedAppRuns = useAtomValue(AppModel.appRunModel.appRuns);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    // Sort app runs: running apps at the top, then by start time (newest first)
    const appRuns = useMemo(() => {
        return [...unsortedAppRuns].sort((a, b) => {
            // First sort by status (running at the top)
            if (a.status === "running" && b.status !== "running") return -1;
            if (a.status !== "running" && b.status === "running") return 1;

            // Then sort by start time (newest first)
            return b.starttime - a.starttime;
        });
    }, [unsortedAppRuns]);

    const handleAppRunClick = (appRunId: string) => {
        AppModel.selectAppRun(appRunId);
        setIsOpen(false); // Close the nav after selection
    };

    const handleClose = () => {
        setIsOpen(false);
    };

    return (
        <>
            {/* Overlay */}
            {isOpen && (
                <div 
                    className="fixed inset-0 bg-black/20 dark:bg-black/50 z-40"
                    onClick={handleClose}
                />
            )}
            
            {/* Left Navigation */}
            <div 
                className={cn(
                    "fixed top-0 left-0 h-full w-64 bg-panel border-r border-border z-50 flex flex-col transition-transform duration-300 ease-in-out",
                    isOpen ? "translate-x-0" : "-translate-x-full"
                )}
            >
                {/* Header with close button */}
                <div className="flex items-center justify-between p-4 border-b border-border">
                    <div className="flex items-center space-x-2">
                        <img src="/outriglogo.svg" alt="Outrig Logo" className="w-[20px] h-[20px]" />
                        <span className="text-primary font-medium">Outrig</span>
                    </div>
                    <button 
                        onClick={handleClose}
                        className="text-secondary hover:text-primary cursor-pointer"
                    >
                        <X size={18} />
                    </button>
                </div>
                
                {/* Navigation Links */}
                <div className="flex-1 overflow-hidden flex flex-col">
                    {/* Top Links */}
                    <div className="p-2">
                        <button 
                            className="w-full flex items-center space-x-2 p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer"
                            onClick={() => {
                                AppModel.selectAppRunsTab();
                                setIsOpen(false);
                            }}
                        >
                            <Home size={16} />
                            <span>Homepage</span>
                        </button>
                    </div>
                    
                    {/* App Runs Section */}
                    <div className="px-4 py-2 text-xs font-medium text-secondary uppercase">
                        App Runs
                    </div>
                    
                    {/* App Runs List (Scrollable) */}
                    <div className="flex-1 overflow-y-auto">
                        {appRuns.length === 0 ? (
                            <div className="px-4 py-2 text-secondary text-sm">No app runs found</div>
                        ) : (
                            <div className="px-2">
                                {appRuns.map((appRun) => (
                                    <div
                                        key={appRun.apprunid}
                                        className={cn(
                                            "p-2 rounded text-sm cursor-pointer",
                                            appRun.apprunid === selectedAppRunId 
                                                ? "bg-buttonhover text-primary" 
                                                : "text-secondary hover:bg-buttonhover hover:text-primary"
                                        )}
                                        onClick={() => handleAppRunClick(appRun.apprunid)}
                                    >
                                        <div className="font-medium truncate">{appRun.appname}</div>
                                        <div className="text-xs text-muted truncate">
                                            {appRun.status === "running" ? "Running" : appRun.status === "done" ? "Done" : "Disconnected"}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                    
                    {/* Bottom Links */}
                    <div className="mt-auto border-t border-border p-2">
                        <button className="w-full flex items-center space-x-2 p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer">
                            <Settings size={16} />
                            <span>Settings</span>
                        </button>
                    </div>
                    
                    {/* Social Links */}
                    <div className="flex justify-center space-x-3 p-4 border-t border-border">
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Github size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <BookOpen size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <MessageSquare size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Youtube size={18} />
                        </a>
                        <a href="#" className="text-secondary hover:text-primary cursor-pointer">
                            <Twitter size={18} />
                        </a>
                    </div>
                </div>
            </div>
        </>
    );
};
