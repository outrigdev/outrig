// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import { Bot, Check, Copy } from "lucide-react";
import { useState } from "react";

const AI_INSTRUCTIONS = `To set up Outrig in your Go application, follow these steps:

1. **Import Outrig SDK**

Add the following import statement to your Go application's main file:

\`\`\`go
import "github.com/outrigdev/outrig"
\`\`\`

2. **Initialize Outrig**

Inside your \`main()\` function, initialize Outrig with your application's name:

\`\`\`go
func main() {
    outrig.Init("your-app-name", nil)

    // Optional: Ensure Outrig is signaled upon application shutdown
    defer outrig.AppDone()

    // Your application code here...
}
\`\`\`

3. **Update Go Modules**

Run the following command to automatically add the Outrig dependency to your \`go.mod\` and \`go.sum\` files:

\`\`\`sh
go mod tidy
\`\`\`

Once you start your application, it will automatically appear in your Outrig dashboard at [http://localhost:5005](http://localhost:5005).`;

interface GettingStartedContentProps {
    hideTitle?: boolean;
    hideFooterText?: boolean;
}

export const GettingStartedContent: React.FC<GettingStartedContentProps> = ({ hideTitle = false, hideFooterText = false }) => {
    const [copied, setCopied] = useState(false);

    const handleCopyInstructions = async () => {
        try {
            await navigator.clipboard.writeText(AI_INSTRUCTIONS);
            setCopied(true);
            setTimeout(() => {
                setCopied(false);
            }, 2000);
        } catch (error) {
            console.error("Failed to copy text:", error);
        }
    };

    const codeWithColorizedComments = (
        <>
            <span className="text-accent">// Step 1: Import the package</span>
            <br />
            import "github.com/outrigdev/outrig"
            <br />
            <br />
            func main() {"{"}
            <br />
            {"    "}
            <span className="text-accent">// Step 2: Initialize Outrig (set your application name)</span>
            <br />
            {"    "}outrig.Init("app-name", nil)
            <br />
            {"    "}
            <br />
            {"    "}
            <span className="text-accent">// Step 3: Optionally signal graceful shutdown</span>
            <br />
            {"    "}defer outrig.AppDone()
            <br />
            {"    "}
            <br />
            {"    "}
            <span className="text-accent">// Your application code here...</span>
            <br />
            {"}"}
        </>
    );

    return (
        <div className="px-6 py-6 max-w-xl w-full mx-auto">
            {!hideTitle && (
                <h3 className="text-primary text-lg font-medium mb-4">Getting Started</h3>
            )}
            <p className="text-secondary mb-6">To connect your Go server or application, follow these steps:</p>
            <div className="bg-black/4 py-5 w-full border-l-2 border-accentbg">
                <div className="px-5">
                    <pre className="whitespace-pre text-left text-sm text-primary overflow-auto w-full">
                        <code>{codeWithColorizedComments}</code>
                    </pre>
                </div>
            </div>
            <div className="mt-8 w-full">
                <div className="bg-panel py-4 border-l-2 border-accentbg">
                    <div className="px-5 text-left">
                        <div className="flex items-center mb-2">
                            <div className="text-accent mr-2">
                                <Bot size={18} />
                            </div>
                            <h4 className="text-primary font-medium">AI Instructions</h4>
                        </div>
                        <p className="text-secondary text-sm mb-3">
                            Using AI? Copy these setup instructions to share with your AI assistant.
                        </p>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={handleCopyInstructions}
                                className={cn(
                                    "p-1 rounded transition-colors cursor-pointer text-primary hover:text-primary/80",
                                    copied && "text-success hover:text-success/80"
                                )}
                                aria-label={copied ? "Copied" : "Copy to clipboard"}
                            >
                                {copied ? <Check size={16} /> : <Copy size={16} />}
                            </button>
                            <span
                                className="text-accent text-sm cursor-pointer"
                                onClick={handleCopyInstructions}
                            >
                                Copy Instructions for AI
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            {!hideFooterText && (
                <p className="text-secondary mt-6">
                    Once you run your application, it will appear here automatically.
                </p>
            )}
        </div>
    );
};