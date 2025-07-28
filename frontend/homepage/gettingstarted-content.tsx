// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { CopyButton } from "@/elements/copybutton";
import { ExternalLink } from "lucide-react";
import { useState } from "react";

interface GettingStartedContentProps {
    hideTitle?: boolean;
    hideFooterText?: boolean;
}

export const GettingStartedContent: React.FC<GettingStartedContentProps> = ({
    hideTitle = false,
    hideFooterText = false,
}) => {
    const [showImportMethod, setShowImportMethod] = useState(false);
    const runCommand = "outrig run main.go";
    const importLine = 'import _ "github.com/outrigdev/outrig/autoinit"';

    const handleCopyRun = async () => {
        await navigator.clipboard.writeText(runCommand);
    };

    const handleCopyImport = async () => {
        await navigator.clipboard.writeText(importLine);
    };

    const runCodeWithComments = (
        <>
            <span className="text-accent"># Run your Go application with Outrig:</span>
            <br />
            <span className="text-accent select-none">&gt; </span>
            {runCommand}
        </>
    );

    const importCodeWithComments = (
        <>
            <span className="text-accent">// Add this import to your main Go file:</span>
            <br />
            {importLine}
            <br />
            <br />
            <span className="text-accent">// That's it! Your app will appear in Outrig when run</span>
        </>
    );

    return (
        <div className="px-6 py-6 max-w-xl w-full mx-auto">
            {!hideTitle && <h3 className="text-primary text-lg font-medium mb-4">Getting Started</h3>}

            {!showImportMethod ? (
                <>
                    <p className="text-secondary mb-6">
                        The easiest way to get started is with the outrig run command:
                    </p>
                    <div className="bg-black/4 py-5 w-full border-l-2 border-accentbg relative">
                        <div className="absolute top-3 right-3">
                            <CopyButton onCopy={handleCopyRun} />
                        </div>
                        <div className="px-5">
                            <pre className="whitespace-pre text-left text-sm text-primary overflow-auto w-full">
                                <code>{runCodeWithComments}</code>
                            </pre>
                        </div>
                    </div>

                    <div className="mt-4">
                        <div className="text-secondary text-sm space-y-1">
                            <div>
                                • Prefer to integrate directly?{" "}
                                <button
                                    onClick={() => setShowImportMethod(true)}
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer"
                                >
                                    Use the import method
                                </button>
                            </div>
                            <div>
                                • Using Docker? See{" "}
                                <a
                                    href="https://outrig.run/docs/using-docker?ref=app"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer inline-flex items-center"
                                >
                                    Docker setup
                                    <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                                </a>
                            </div>
                            <div>
                                • Need more control? See{" "}
                                <a
                                    href="https://outrig.run/docs/advanced-configuration?ref=app"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer inline-flex items-center"
                                >
                                    advanced configuration
                                    <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                                </a>
                            </div>
                        </div>
                    </div>
                </>
            ) : (
                <>
                    <p className="text-secondary mb-6">
                        To integrate Outrig directly into your Go application, add this import:
                    </p>
                    <div className="bg-black/4 py-5 w-full border-l-2 border-accentbg relative">
                        <div className="absolute top-3 right-3">
                            <CopyButton onCopy={handleCopyImport} />
                        </div>
                        <div className="px-5">
                            <pre className="whitespace-pre text-left text-sm text-primary overflow-auto w-full">
                                <code>{importCodeWithComments}</code>
                            </pre>
                        </div>
                    </div>

                    <div className="mt-4">
                        <div className="text-secondary text-sm space-y-1">
                            <div>
                                • Prefer the command line?{" "}
                                <button
                                    onClick={() => setShowImportMethod(false)}
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer"
                                >
                                    Use <span className="font-mono">`outrig run`</span> instead
                                </button>
                            </div>
                            <div>
                                • Using Docker? See{" "}
                                <a
                                    href="https://outrig.run/docs/using-docker?ref=app"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer inline-flex items-center"
                                >
                                    Docker setup
                                    <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                                </a>
                            </div>
                            <div>
                                • Need more control? See{" "}
                                <a
                                    href="https://outrig.run/docs/advanced-configuration?ref=app"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-accent hover:text-accent-hover hover:underline cursor-pointer inline-flex items-center"
                                >
                                    advanced configuration
                                    <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                                </a>
                            </div>
                        </div>
                    </div>
                </>
            )}

            {!hideFooterText && (
                <p className="text-secondary mt-6">Once you run your application, it will appear here automatically.</p>
            )}
        </div>
    );
};
