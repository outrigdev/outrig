import { useAtomValue } from "jotai";
import { useEffect } from "react";
import { AppModel } from "../appmodel";

export const AppRunList: React.FC = () => {
    const appRuns = useAtomValue(AppModel.appRuns);
    
    useEffect(() => {
        // Load app runs when the component mounts
        AppModel.loadAppRuns();
        
        // Set up a refresh interval
        const intervalId = setInterval(() => {
            AppModel.loadAppRuns();
        }, 5000); // Refresh every 5 seconds
        
        return () => clearInterval(intervalId);
    }, []);
    
    const formatTimestamp = (timestamp: number): string => {
        const date = new Date(timestamp);
        return date.toLocaleString();
    };
    
    const handleAppRunClick = (appRunId: string) => {
        AppModel.selectAppRun(appRunId);
    };
    
    return (
        <div className="w-full h-full flex flex-col">
            <div className="py-2 px-4 border-b border-border">
                <h2 className="text-lg font-semibold text-primary">App Runs</h2>
            </div>
            
            <div className="flex-1 overflow-auto">
                {appRuns.length === 0 ? (
                    <div className="flex items-center justify-center h-full text-secondary">
                        No app runs found
                    </div>
                ) : (
                    <div className="divide-y divide-border">
                        {appRuns.map((appRun) => (
                            <div 
                                key={appRun.apprunid}
                                className="p-4 hover:bg-buttonhover cursor-pointer"
                                onClick={() => handleAppRunClick(appRun.apprunid)}
                            >
                                <div className="flex justify-between items-center">
                                    <div className="font-medium text-primary">{appRun.appname}</div>
                                    <div className="text-xs text-secondary">
                                        {appRun.status === "running" ? (
                                            <span className="px-2 py-1 rounded-full bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                                                Running
                                            </span>
                                        ) : appRun.status === "done" ? (
                                            <span className="px-2 py-1 rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                                                Done
                                            </span>
                                        ) : (
                                            <span className="px-2 py-1 rounded-full bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                                                Disconnected
                                            </span>
                                        )}
                                    </div>
                                </div>
                                <div className="mt-1 text-sm text-secondary">
                                    Started: {formatTimestamp(appRun.starttime)}
                                </div>
                                <div className="mt-1 text-xs text-muted">
                                    ID: {appRun.apprunid}
                                </div>
                                <div className="mt-1 text-xs text-muted">
                                    Logs: {appRun.numlogs}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
