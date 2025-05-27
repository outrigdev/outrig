// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { DefaultRpcClient } from "@/init";
import { RpcApi } from "@/rpc/rpcclientapi";
import { useAtomValue } from "jotai";
import { Copy, Download, ExternalLink } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Modal } from "./modal";

// Internal component that renders the actual modal content
const UpdateModal = () => {
    const newerVersion = useAtomValue(AppModel.newerVersion);
    const fromTrayApp = useAtomValue(AppModel.fromTrayApp);
    const [platform, setPlatform] = useState<"mac" | "linux" | "other">("other");
    const [isUpdating, setIsUpdating] = useState(false);

    useEffect(() => {
        // Detect platform
        const userAgent = navigator.userAgent.toLowerCase();
        if (userAgent.includes("mac")) {
            setPlatform("mac");
        } else if (userAgent.includes("linux")) {
            setPlatform("linux");
        } else {
            setPlatform("other");
        }
    }, []);

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        AppModel.showToast("Copied", "Command copied to clipboard", 2000);
    };

    const handleTrayUpdate = async () => {
        if (!DefaultRpcClient) return;

        setIsUpdating(true);
        try {
            await RpcApi.TriggerTrayUpdateCommand(DefaultRpcClient);
            AppModel.showToast("Update Started", "The update process has been initiated", 3000);
            AppModel.closeUpdateModal();
        } catch (err) {
            console.error("Failed to trigger tray update:", err);
            AppModel.showToast("Update Failed", "Failed to start the update process", 3000);
        } finally {
            setIsUpdating(false);
        }
    };

    return (
        <Modal isOpen={true} title={`Update Available: ${newerVersion}`} onClose={() => AppModel.closeUpdateModal()}>
            <div className="flex flex-col gap-6">
                {fromTrayApp ? (
                    <>
                        <p className="text-primary">
                            A new version of Outrig is available. Click the button below to launch the updater.
                        </p>
                        <button
                            onClick={handleTrayUpdate}
                            disabled={isUpdating}
                            className="flex items-center justify-center gap-2 bg-accent hover:bg-accent/80 disabled:bg-accent/50 text-white px-4 py-2 rounded-md transition-colors cursor-pointer disabled:cursor-not-allowed"
                        >
                            <Download size={16} />
                            {isUpdating ? "Installing Update..." : "Install Update Now"}
                        </button>
                    </>
                ) : (
                    <>
                        <p className="text-primary">
                            A new version of Outrig is available. You can update using the instructions below.
                        </p>

                        {platform === "mac" && (
                            <div className="flex flex-col gap-4">
                                <h3 className="text-lg font-medium text-primary">MacOS (Intel, Apple Silicon)</h3>
                                <div className="bg-gray-100 dark:bg-gray-800 rounded-md p-3 relative">
                                    <pre className="text-sm text-primary font-mono">
                                        brew install outrigdev/outrig/outrig
                                    </pre>
                                    <button
                                        onClick={() => copyToClipboard("brew install outrigdev/outrig/outrig")}
                                        className="absolute top-2 right-2 text-secondary hover:text-primary cursor-pointer"
                                        aria-label="Copy command"
                                    >
                                        <Copy size={16} />
                                    </button>
                                </div>
                            </div>
                        )}

                        {platform === "linux" && (
                            <div className="flex flex-col gap-4">
                                <h3 className="text-lg font-medium text-primary">Linux (x64, arm64)</h3>
                                <div className="bg-gray-100 dark:bg-gray-800 rounded-md p-3 relative">
                                    <pre className="text-sm text-primary font-mono">
                                        curl -sf https://outrig.run/install.sh | sh
                                    </pre>
                                    <button
                                        onClick={() => copyToClipboard("curl -sf https://outrig.run/install.sh | sh")}
                                        className="absolute top-2 right-2 text-secondary hover:text-primary cursor-pointer"
                                        aria-label="Copy command"
                                    >
                                        <Copy size={16} />
                                    </button>
                                </div>
                                <p className="text-sm text-secondary">
                                    This script installs to ~/.local/bin without requiring root access.
                                </p>
                            </div>
                        )}

                        {/* Show GitHub releases link for all platforms */}
                        <div className="flex flex-col gap-2 mt-2">
                            <a
                                href="https://github.com/outrigdev/outrig/releases"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-2 text-accent hover:text-accent/80 transition-colors cursor-pointer"
                            >
                                <ExternalLink size={16} />
                                <span>View all releases on GitHub</span>
                            </a>
                        </div>
                    </>
                )}
            </div>
        </Modal>
    );
};

// Container component that handles the open/close state
export const UpdateModalContainer = React.memo(function UpdateModalContainer() {
    const isUpdateModalOpen = useAtomValue(AppModel.updateModalOpen);
    if (!isUpdateModalOpen) return null;
    return <UpdateModal />;
});

UpdateModalContainer.displayName = "UpdateModalContainer";
