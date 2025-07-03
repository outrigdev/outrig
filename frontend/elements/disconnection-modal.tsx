// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import {
    serverConnectedAtom,
    serverConnectionStateAtom,
    serverReconnectAttemptsAtom,
    WebSocketController,
} from "@/websocket/client";
import { useAtomValue } from "jotai";
import { RefreshCw, WifiOff } from "lucide-react";
import React, { useEffect } from "react";

interface DisconnectionModalProps {
    isOpen: boolean;
    connectionState: "connecting" | "connected" | "failed";
}

export const DisconnectionModal: React.FC<DisconnectionModalProps> = ({ isOpen, connectionState }) => {
    const reconnectAttempts = useAtomValue(serverReconnectAttemptsAtom);

    const handleRetryConnection = () => {
        WebSocketController.getInstance()?.retryConnection();
    };
    useEffect(() => {
        if (isOpen) {
            // Prevent escape key from closing the modal
            const handleKeyDown = (e: KeyboardEvent) => {
                if (e.key === "Escape") {
                    e.preventDefault();
                    e.stopPropagation();
                }
            };

            document.addEventListener("keydown", handleKeyDown, true);

            return () => {
                document.removeEventListener("keydown", handleKeyDown, true);
            };
        }
    }, [isOpen]);

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 flex items-center justify-center z-[60]">
            {/* Backdrop - no onClick handler to prevent dismissal */}
            <div className="absolute inset-0 bg-[#000000]/50" aria-hidden="true"></div>

            {/* Modal content - non-dismissable */}
            <div
                className={cn(
                    "bg-panel border border-border rounded-md shadow-lg w-[400px] max-w-[90vw] flex flex-col focus:outline-none z-10"
                )}
                tabIndex={-1}
                role="dialog"
                aria-modal="true"
            >
                {/* Header - no close button */}
                <div className="flex items-center px-4 py-3 border-b border-border">
                    <WifiOff size={20} className="text-error mr-3" />
                    <h2 className="text-primary font-bold">Server Disconnected</h2>
                </div>

                {/* Content */}
                <div className="p-4">
                    <p className="text-muted mb-4">
                        The connection to the Outrig server has been lost. Please check that the server is running.
                    </p>

                    {connectionState === "connecting" && (
                        <div className="flex items-center justify-center">
                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
                            <span className="ml-2 text-muted">
                                Attempting to reconnect{".".repeat(reconnectAttempts)}
                            </span>
                        </div>
                    )}

                    {connectionState === "failed" && (
                        <div className="flex flex-col items-center gap-3">
                            <button
                                onClick={handleRetryConnection}
                                className={cn(
                                    "flex items-center gap-2 px-4 py-2 bg-accent text-white rounded-md",
                                    "hover:bg-accentbg transition-colors cursor-pointer"
                                )}
                            >
                                <RefreshCw size={16} />
                                Retry Connection
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

// Container component
export const DisconnectionModalContainer: React.FC = () => {
    const isServerConnected = useAtomValue(serverConnectedAtom);
    const connectionState = useAtomValue(serverConnectionStateAtom);
    return <DisconnectionModal isOpen={!isServerConnected} connectionState={connectionState} />;
};
