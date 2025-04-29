// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import { Copy, ExternalLink } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Modal } from "./modal";

// Internal component that renders the actual modal content
const UpdateModal = () => {
    const newerVersion = useAtomValue(AppModel.newerVersion);
    const [platform, setPlatform] = useState<"mac" | "linux" | "other">("other");

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

    return (
        <Modal isOpen={true} title={`Update Available: ${newerVersion}`} onClose={() => AppModel.closeUpdateModal()}>
            <div className="flex flex-col gap-6">
                <p className="text-primary">
                    A new version of Outrig is available. You can update using the instructions below.
                </p>

                {platform === "mac" && (
                    <div className="flex flex-col gap-4">
                        <h3 className="text-lg font-medium text-primary">MacOS (Intel, Apple Silicon)</h3>
                        <div className="bg-gray-100 dark:bg-gray-800 rounded-md p-3 relative">
                            <pre className="text-sm text-primary font-mono">brew install outrigdev/outrig/outrig</pre>
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
