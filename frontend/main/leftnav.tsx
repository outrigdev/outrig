import { useAtom, useAtomValue } from "jotai";
import { BookOpen, Clock, Github, Home, MessageSquare, Settings, Twitter, X, Youtube } from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { AppModel } from "../appmodel";
import { cn, formatDuration, formatRelativeTime } from "../util/util";

// AppRunItem component for displaying a single app run item
interface AppRunItemProps {
    appRun: AppRunInfo;
    isSelected: boolean;
    onClick: (appRunId: string) => void;
}

export const AppRunItem: React.FC<AppRunItemProps> = ({ appRun, isSelected, onClick }) => {
    const [currentTime, setCurrentTime] = useState(Date.now());

    // Only update the time for running apps
    useEffect(() => {
        if (appRun.status === "running") {
            const interval = setInterval(() => {
                setCurrentTime(Date.now());
            }, 1000);
            return () => clearInterval(interval);
        }
    }, [appRun.status]);
    return (
        <div
            className={cn(
                "p-2 rounded text-sm cursor-pointer",
                isSelected ? "bg-buttonhover text-primary" : "text-secondary hover:bg-buttonhover hover:text-primary"
            )}
            onClick={() => onClick(appRun.apprunid)}
        >
            <div className="font-medium flex items-center justify-between">
                <div className="flex items-center overflow-hidden">
                    <span className="overflow-hidden text-ellipsis whitespace-nowrap">{appRun.appname}</span>
                    <span className="text-[10px] ml-1 whitespace-nowrap text-muted">
                        ({appRun.apprunid.substring(0, 4)})
                    </span>
                </div>
                {appRun.status === "running" && <div className="w-2 h-2 rounded-full bg-green-500 ml-1"></div>}
            </div>
            <div className="text-xs text-muted truncate ml-2 flex items-center">
                <span className="inline-block w-16">
                    {appRun.status === "running" ? "Running" : formatRelativeTime(appRun.starttime)}
                </span>
                <Clock size={12} className="mr-1" />
                {appRun.status === "running"
                    ? formatDuration(Math.floor((currentTime - appRun.starttime) / 1000))
                    : formatDuration(Math.floor((appRun.lastmodtime - appRun.starttime) / 1000))}
            </div>
        </div>
    );
};

// AppRunList component for displaying the list of app runs in the left navigation
export const AppRunList: React.FC = () => {
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

    return (
        <>
            <div className="px-4 pt-2 pb-1 text-[10px] font-bold text-secondary uppercase">App Runs</div>

            {/* App Runs List (Scrollable) */}
            <div className="flex-1 overflow-y-auto">
                {appRuns.length === 0 ? (
                    <div className="px-4 py-2 text-secondary text-sm">No app runs found</div>
                ) : (
                    <div className="pl-3 pr-2">
                        {appRuns.map((appRun) => (
                            <AppRunItem
                                key={appRun.apprunid}
                                appRun={appRun}
                                isSelected={appRun.apprunid === selectedAppRunId}
                                onClick={handleAppRunClick}
                            />
                        ))}
                    </div>
                )}
            </div>
        </>
    );
};

export const LeftNav: React.FC = () => {
    const [isOpen, setIsOpen] = useAtom(AppModel.leftNavOpen);

    const handleClose = () => {
        setIsOpen(false);
    };

    return (
        <>
            {/* Overlay */}
            {isOpen && <div className="fixed inset-0 bg-black/20 backdrop-blur-[1px] z-40" onClick={handleClose} />}

            {/* Left Navigation */}
            <div
                className={cn(
                    "fixed top-0 left-0 h-full w-64 bg-panel border-r-2 border-border z-50 flex flex-col transition-transform duration-300 ease-in-out",
                    isOpen ? "translate-x-0" : "-translate-x-full"
                )}
            >
                {/* Header with close button */}
                <div
                    className="flex items-center justify-between p-4 border-b border-border cursor-pointer"
                    onClick={() => setIsOpen(false)}
                >
                    <div className="flex items-center space-x-2">
                        <img src="/outriglogo.svg" alt="Outrig Logo" className="w-[20px] h-[20px]" />
                        <span className="text-primary font-medium">Outrig</span>
                    </div>
                    <button
                        onClick={(e) => {
                            e.stopPropagation();
                            handleClose();
                        }}
                        className="text-secondary hover:text-primary cursor-pointer"
                    >
                        <X size={18} />
                    </button>
                </div>

                {/* Navigation Links */}
                <div className="flex-1 overflow-hidden flex flex-col">
                    {/* Top Links */}
                    <div className="p-2 border-b border-border">
                        <button
                            className="w-full flex items-center space-x-2 p-2 text-secondary hover:text-primary hover:bg-buttonhover rounded cursor-pointer"
                            onClick={() => {
                                AppModel.selectAppRunsTab();
                                setIsOpen(false);
                            }}
                        >
                            <Home size={16} />
                            <span>Home</span>
                        </button>
                    </div>

                    {/* App Runs Section */}
                    <AppRunList />

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
