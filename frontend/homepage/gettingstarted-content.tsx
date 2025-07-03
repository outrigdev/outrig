// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { CopyButton } from "@/elements/copybutton";
import { ExternalLink } from "lucide-react";

interface GettingStartedContentProps {
    hideTitle?: boolean;
    hideFooterText?: boolean;
}

export const GettingStartedContent: React.FC<GettingStartedContentProps> = ({
    hideTitle = false,
    hideFooterText = false,
}) => {
    const importLine = 'import _ "github.com/outrigdev/outrig/autoinit"';

    const handleCopyImport = async () => {
        await navigator.clipboard.writeText(importLine);
    };

    const codeWithColorizedComments = (
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
            <p className="text-secondary mb-6">To connect your Go server or application, add this import:</p>
            <div className="bg-black/4 py-5 w-full border-l-2 border-accentbg relative">
                <div className="absolute top-3 right-3">
                    <CopyButton onCopy={handleCopyImport} />
                </div>
                <div className="px-5">
                    <pre className="whitespace-pre text-left text-sm text-primary overflow-auto w-full">
                        <code>{codeWithColorizedComments}</code>
                    </pre>
                </div>
            </div>

            <div className="mt-4">
                <div className="text-secondary text-sm space-y-1">
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

            {!hideFooterText && (
                <p className="text-secondary mt-6">Once you run your application, it will appear here automatically.</p>
            )}
        </div>
    );
};
