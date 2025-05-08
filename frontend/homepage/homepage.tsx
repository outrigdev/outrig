// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AppRunList } from "@/apprunlist/apprunlist";
import { SettingsButton } from "@/elements/settingsbutton";
import { UpdateBadge } from "@/elements/updatebadge";
import { StatusBar } from "@/mainapp/statusbar";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import { BookText, ExternalLink, Github } from "lucide-react";

const GettingStartedWithOutrig: React.FC = () => {
    // Split the code into parts to apply different styling to comments
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
        <div className="flex flex-col items-center h-full p-6 text-center">
            <div className="grow" />
            <div className="flex flex-col items-center">
                <h3 className="text-primary text-lg font-medium mb-4">Getting Started with Outrig</h3>
                <p className="text-secondary mb-6">To connect your Go server or application, follow these steps:</p>
                <pre className="whitespace-pre bg-panel border border-border rounded-lg p-4 text-left text-sm text-primary overflow-auto w-full max-w-xl">
                    <code>{codeWithColorizedComments}</code>
                </pre>
                <p className="text-secondary mt-6">Once you run your application, it will appear here automatically.</p>
            </div>
            <div className="grow-2" />
        </div>
    );
};

const AppRunSelectionColumn: React.FC<{ hasAppRuns: boolean }> = ({ hasAppRuns }) => {
    return (
        <div
            className={cn(
                "border-t md:border-t-0 border-border",
                "flex flex-col h-full overflow-hidden",
                hasAppRuns ? "md:w-[500px]" : "md:flex-grow"
            )}
        >
            <div className={cn("p-4 bg-panel border-b border-border", !hasAppRuns ? "pl-6" : null)}>
                {hasAppRuns ? (
                    <>
                        <h2 className="text-primary text-xl font-medium">Select a Run</h2>
                        <p className="text-secondary text-sm mt-1">
                            Choose run from the list to explore details and insights.
                        </p>
                    </>
                ) : (
                    <>
                        <h2 className="text-primary text-xl font-medium">Waiting for Connection...</h2>
                        <p className="text-secondary text-sm mt-3">
                            Your connected server or application runs will appear here automatically.
                        </p>
                    </>
                )}
            </div>
            <div className="flex-grow overflow-auto">
                <AppRunList emptyStateComponent={<GettingStartedWithOutrig />} />
            </div>
        </div>
    );
};

const WelcomeColumn: React.FC = () => {
    return (
        <div className="md:flex-grow flex flex-col md:border-l border-border">
            <div className="grow"></div>
            <div className="max-w-xl mx-auto p-8 flex flex-col items-center">
                {/* Logo */}
                <div className="mb-6">
                    <img src="/outriglogo.svg" alt="Outrig Logo" className="w-16 h-16" />
                </div>

                <div className="text-center mb-8">
                    <h1 className="text-primary text-3xl font-medium mb-4">Welcome to Outrig!</h1>
                    <p className="text-secondary text-sm">
                        Outrig gives you visibility into your running Go servers and applications, helping you quickly
                        identify issues and optimize performance.
                    </p>
                </div>

                {/* Cards container - stacked layout */}
                <div className="w-full flex flex-col gap-6">
                    {/* GitHub section */}
                    <div className="bg-panel py-5 w-full">
                        <div className="border-l-2 border-accentbg px-5">
                            <div className="flex items-center mb-2">
                                <div className="text-accent mr-2">
                                    <Github size={20} />
                                </div>
                                <h3 className="text-primary font-medium">GitHub</h3>
                            </div>
                            <p className="text-secondary text-sm mb-2">Like Outrig? Give us a star on GitHub!</p>
                            <a
                                href="https://github.com/outrigdev/outrig"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-accent hover:text-accent-hover hover:underline text-sm flex items-center cursor-pointer truncate"
                            >
                                github.com/outrigdev/outrig
                                <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                            </a>
                        </div>
                    </div>

                    {/* Documentation section */}
                    <div className="bg-panel py-5 w-full">
                        <div className="border-l-2 border-accentbg px-5">
                            <div className="flex items-center mb-2">
                                <div className="text-accent mr-2">
                                    <BookText size={20} />
                                </div>
                                <h3 className="text-primary font-medium">Documentation</h3>
                            </div>
                            <p className="text-secondary text-sm mb-2">Learn how to get the most out of Outrig</p>
                            <a
                                href="https://outrig.run/docs/"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-accent hover:text-accent-hover hover:underline text-sm flex items-center cursor-pointer truncate"
                            >
                                outrig.run/docs/
                                <ExternalLink size={14} className="ml-1 flex-shrink-0" />
                            </a>
                        </div>
                    </div>
                </div>
            </div>
            <div className="grow-[2]"></div>
        </div>
    );
};

const LeftColumn: React.FC = () => {
    return (
        <div className="hidden md:block w-[50px] h-full bg-gradient-to-b from-accent/20 to-accent/5">
            {/* This column is intentionally left empty for visual design purposes */}
        </div>
    );
};

export const HomePage: React.FC = () => {
    const appRunCount = useAtomValue(AppModel.appRunModel.appRunCount);
    const hasAppRuns = appRunCount > 0;
    const isDarkMode = useAtomValue(AppModel.darkMode);
    return (
        <div className="flex h-screen w-full">
            {/* Left accent column - only shown when there are app runs */}
            {hasAppRuns && (
                <div className="hidden md:block w-[50px] h-full bg-gradient-to-b from-accent/20 to-accent/5" />
            )}

            {/* Main content container */}
            <div className="flex flex-col flex-grow h-full overflow-hidden">
                {/* Header */}
                <header className="bg-panel border-b border-border p-4 flex items-center justify-between">
                    <div className="flex items-center">
                        <img
                            src={isDarkMode ? "/logo-dark.png" : "/logo-light.png"}
                            alt="Outrig Logo"
                            className="h-8"
                        />
                    </div>
                    <div className="flex items-end self-end">
                        <SettingsButton onClick={() => AppModel.openSettingsModal()} />
                        <UpdateBadge onClick={() => AppModel.openUpdateModal()} />
                    </div>
                </header>

                {/* Main content */}
                <main className="flex-grow flex flex-col md:flex-row overflow-hidden w-full">
                    <AppRunSelectionColumn hasAppRuns={hasAppRuns} />
                    <WelcomeColumn />
                </main>

                {/* Status Bar */}
                <StatusBar />
            </div>
        </div>
    );
};
