// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { DefaultRpcClient } from "@/init";
import { RpcApi } from "@/rpc/rpcclientapi";
import { cn } from "@/util/util";
import { AlertCircle, ExternalLink, Loader2, Play, Square } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type DemoAppState = "unknown" | "not_running" | "launching" | "running" | "error";

interface DemoAppStatus {
    state: DemoAppState;
    error?: string;
}

export const DemoAppController: React.FC = () => {
    const [status, setStatus] = useState<DemoAppStatus>({ state: "unknown" });
    const [isLoading, setIsLoading] = useState(false);
    const [showStoppedMessage, setShowStoppedMessage] = useState(false);
    const demoPopupRef = useRef<Window | null>(null);

    const checkStatus = async () => {
        try {
            const statusStr = await RpcApi.GetDemoAppStatusCommand(DefaultRpcClient);
            if (statusStr === "running") {
                setStatus({ state: "running" });
            } else if (statusStr === "stopped") {
                setStatus({ state: "not_running" });
            } else if (statusStr === "error") {
                setStatus({ state: "error", error: "Demo app encountered an error" });
            } else {
                setStatus({ state: "not_running" });
            }
        } catch (error) {
            console.error("Failed to check demo app status:", error);
            setStatus({ state: "error", error: String(error) });
        }
    };

    const openDemoPopup = () => {
        if (demoPopupRef.current && !demoPopupRef.current.closed) {
            demoPopupRef.current.focus();
            return;
        }

        const popup = window.open(
            "http://localhost:22005",
            "outrig-demo-app",
            "width=1100,height=930,scrollbars=yes,resizable=yes,toolbar=no,menubar=no,location=no,status=no"
        );

        if (popup) {
            demoPopupRef.current = popup;
        }
    };

    const closeDemoPopup = () => {
        if (demoPopupRef.current && !demoPopupRef.current.closed) {
            demoPopupRef.current.close();
            demoPopupRef.current = null;
        }
    };

    const launchDemo = async () => {
        setIsLoading(true);
        setStatus({ state: "launching" });
        try {
            await RpcApi.LaunchDemoAppCommand(DefaultRpcClient);
            await checkStatus();
            // Wait 500ms then open popup if no error occurred
            setTimeout(() => {
                openDemoPopup();
            }, 500);
        } catch (error) {
            console.error("Failed to launch demo app:", error);
            setStatus({ state: "error", error: String(error) });
        } finally {
            setIsLoading(false);
        }
    };

    const killDemo = async () => {
        setIsLoading(true);
        try {
            await RpcApi.KillDemoAppCommand(DefaultRpcClient);
            closeDemoPopup();
            await checkStatus();
            setShowStoppedMessage(true);
            setTimeout(() => {
                setShowStoppedMessage(false);
            }, 1500);
        } catch (error) {
            console.error("Failed to kill demo app:", error);
            setStatus({ state: "error", error: String(error) });
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        checkStatus();
    }, []);

    useEffect(() => {
        let interval: NodeJS.Timeout;

        if (status.state === "running" || status.state === "launching") {
            interval = setInterval(checkStatus, 1000);
        }

        return () => {
            if (interval) {
                clearInterval(interval);
            }
        };
    }, [status.state]);

    const renderButton = () => {
        switch (status.state) {
            case "unknown":
                return (
                    <button
                        disabled
                        className="flex items-center gap-2 px-4 py-2 bg-accent/10 text-accent rounded cursor-not-allowed"
                    >
                        <Loader2 size={16} className="animate-spin" />
                        Checking Status...
                    </button>
                );

            case "not_running":
                if (showStoppedMessage) {
                    return (
                        <div className="flex items-center gap-2 text-success text-sm">
                            <div className="w-2 h-2 bg-success rounded-full" />
                            Demo App Stopped
                        </div>
                    );
                }
                return (
                    <button
                        onClick={launchDemo}
                        disabled={isLoading}
                        className={cn(
                            "flex items-center gap-2 px-4 py-2 rounded transition-colors cursor-pointer",
                            "bg-accent text-white hover:bg-accent-hover",
                            isLoading && "opacity-50 cursor-not-allowed"
                        )}
                    >
                        {isLoading ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                        Launch Demo Application
                    </button>
                );

            case "launching":
                return (
                    <button
                        disabled
                        className="flex items-center gap-2 px-4 py-2 bg-accent/10 text-accent rounded cursor-not-allowed"
                    >
                        <Loader2 size={16} className="animate-spin" />
                        Launching...
                    </button>
                );

            case "running":
                return (
                    <div className="flex flex-col gap-2">
                        <div className="flex items-center gap-2 text-success text-sm">
                            <div className="w-2 h-2 bg-success rounded-full animate-pulse" />
                            Demo application is running
                        </div>
                        <div className="flex gap-2">
                            <button
                                onClick={openDemoPopup}
                                className="flex items-center gap-2 px-3 py-2 rounded transition-colors cursor-pointer border border-accent/50 text-accent hover:bg-accent/10"
                            >
                                <ExternalLink size={14} />
                                Open Demo
                            </button>
                            <button
                                onClick={killDemo}
                                disabled={isLoading}
                                className={cn(
                                    "flex items-center gap-2 px-4 py-2 rounded transition-colors cursor-pointer",
                                    "bg-accent text-white hover:bg-accent-hover",
                                    isLoading && "opacity-50 cursor-not-allowed"
                                )}
                            >
                                {isLoading ? <Loader2 size={16} className="animate-spin" /> : <Square size={16} />}
                                Stop Demo
                            </button>
                        </div>
                    </div>
                );

            case "error":
                return (
                    <div className="flex flex-col gap-2">
                        <div className="flex items-center gap-2 text-error text-sm">
                            <AlertCircle size={16} />
                            Error: {status.error}
                        </div>
                        <button
                            onClick={launchDemo}
                            disabled={isLoading}
                            className={cn(
                                "flex items-center gap-2 px-4 py-2 rounded transition-colors cursor-pointer",
                                "bg-accent text-white hover:bg-accent-hover",
                                isLoading && "opacity-50 cursor-not-allowed"
                            )}
                        >
                            {isLoading ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                            Try Again
                        </button>
                    </div>
                );

            default:
                return null;
        }
    };

    return (
        <div className="bg-panel py-5 w-full">
            <div className="border-l-2 border-accentbg px-5">
                <div className="flex items-center mb-2">
                    <div className="text-accent mr-2">
                        <Play size={20} />
                    </div>
                    <h3 className="text-primary font-medium">Demo Application</h3>
                </div>
                <p className="text-secondary text-sm mb-3">
                    Try Outrig with a sample Go application that generates logs and goroutines
                </p>
                {renderButton()}
            </div>
        </div>
    );
};
