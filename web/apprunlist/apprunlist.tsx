import { useAtomValue } from "jotai";
import { useMemo } from "react";
import { AppModel } from "../appmodel";
import { Tag } from "../elements/tag";
import { cn } from "../util/util";

const formatTimestamp = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString();
};

interface AppRunStatusTagProps {
    status: string;
}

const AppRunStatusTag: React.FC<AppRunStatusTagProps> = ({ status }) => {
    if (status === "running") {
        return <Tag label="Running" variant="success" isSelected={true} />;
    } else if (status === "done") {
        return <Tag label="Done" variant="info" isSelected={true} />;
    } else {
        return <Tag label="Disconnected" variant="secondary" isSelected={true} />;
    }
};

interface AppRunItemProps {
    appRun: AppRunInfo;
    onClick: (appRunId: string) => void;
    isSelected: boolean;
}

const AppRunItem: React.FC<AppRunItemProps> = ({ appRun, onClick, isSelected }) => {
    return (
        <div
            className={cn(
                "p-4 hover:bg-buttonhover cursor-pointer",
                isSelected && "bg-buttonhover border-l-4 border-l-accent"
            )}
            onClick={() => onClick(appRun.apprunid)}
        >
            <div className="flex justify-between items-center">
                <div className="font-medium text-primary">{appRun.appname}</div>
                <div className="text-xs text-secondary">
                    <AppRunStatusTag status={appRun.status} />
                </div>
            </div>
            <div className="mt-1 text-sm text-secondary">Started: {formatTimestamp(appRun.starttime)}</div>
            <div className="mt-1 text-xs text-muted">ID: {appRun.apprunid}</div>
            <div className="mt-1 text-xs text-muted">Logs: {appRun.numlogs}</div>
        </div>
    );
};

const NoAppRunsFound: React.FC = () => {
    return <div className="flex items-center justify-center h-full text-secondary">No app runs found</div>;
};

export const AppRunList: React.FC = () => {
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
    };

    return (
        <div className="w-full h-full flex flex-col">
            <div className="flex-1 overflow-auto">
                {appRuns.length === 0 ? (
                    <NoAppRunsFound />
                ) : (
                    <div className="divide-y divide-border">
                        {appRuns.map((appRun) => (
                            <AppRunItem
                                key={appRun.apprunid}
                                appRun={appRun}
                                onClick={handleAppRunClick}
                                isSelected={appRun.apprunid === selectedAppRunId}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
