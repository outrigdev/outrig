// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { AppRunList } from "@/apprunlist/apprunlist";
import { SettingsButton } from "@/elements/settingsbutton";
import { UpdateBadge } from "@/elements/updatebadge";
import { StatusBar } from "@/mainapp/statusbar";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import { ExternalLink, Github } from "lucide-react";

export const HomePage: React.FC = () => {
    const appRunCount = useAtomValue(AppModel.appRunModel.appRunCount);
    const hasAppRuns = appRunCount > 0;
    const isDarkMode = useAtomValue(AppModel.darkMode);
    return (
        <>
            {/* Header */}
            <header className="bg-panel border-b border-border p-4 flex items-center justify-between">
                <div className="flex items-center">
                    <img src={isDarkMode ? "/logo-dark.png" : "/logo-light.png"} alt="Outrig Logo" className="h-8" />
                </div>
                <div className="flex items-end self-end">
                    <SettingsButton onClick={() => AppModel.openSettingsModal()} />
                    <UpdateBadge onClick={() => AppModel.openUpdateModal()} />
                </div>
            </header>

            {/* Main content */}
            <main className="flex-grow flex flex-col md:flex-row overflow-hidden">
                {/* Welcome section */}
                <div className="w-full md:w-1/2 flex flex-col">
                    <div className="grow"></div>
                    <div className="max-w-md mx-auto p-8">
                        <div className="text-center mb-8">
                            <h1 className="text-primary text-3xl font-medium mb-4">Welcome to Outrig!</h1>
                        </div>

                        {/* GitHub section */}
                        <div className="bg-panel border border-border rounded-lg p-6 mb-6">
                            <div className="flex items-start">
                                <div className="text-accent mr-4">
                                    <Github size={24} className="cursor-pointer" />
                                </div>
                                <div>
                                    <h3 className="text-primary text-lg font-medium mb-2">Support us on GitHub</h3>
                                    <p className="text-secondary text-sm mb-3">
                                        Outrig is open-source, runs 100% locally, and no application data ever leaves
                                        your machine. Please show your support by giving us a star!
                                    </p>
                                    <a
                                        href="https://github.com/outrigdev/outrig"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-accent hover:text-accent-hover text-sm flex items-center cursor-pointer"
                                    >
                                        github.com/outrigdev/outrig
                                        <ExternalLink size={14} className="ml-1" />
                                    </a>
                                </div>
                            </div>
                        </div>

                        {/* Documentation section */}
                        <div className="bg-panel border border-border rounded-lg p-6">
                            <div className="flex items-start">
                                <div className="text-accent mr-4">
                                    <ExternalLink size={24} className="cursor-pointer" />
                                </div>
                                <div>
                                    <h3 className="text-primary text-lg font-medium mb-2">Documentation</h3>
                                    <p className="text-secondary text-sm mb-3">
                                        Check out our documentation to learn how to get the most out of Outrig.
                                    </p>
                                    <a
                                        href="https://outrig.run/docs/"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-accent hover:text-accent-hover text-sm flex items-center cursor-pointer"
                                    >
                                        outrig.run/docs/
                                        <ExternalLink size={14} className="ml-1" />
                                    </a>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="grow-[2]"></div>
                </div>

                {/* App run selection */}
                <div
                    className={cn(
                        "w-full md:w-1/2 border-t md:border-t-0 md:border-l border-border",
                        "flex flex-col h-full overflow-hidden"
                    )}
                >
                    <div className="p-6 bg-panel border-b border-border">
                        {hasAppRuns ? (
                            <>
                                <h2 className="text-primary text-xl font-medium">Select a Run</h2>
                                <p className="text-secondary text-sm mt-1">
                                    Choose an active or completed session to start debugging.
                                </p>
                            </>
                        ) : (
                            <>
                                <h2 className="text-primary text-xl font-medium">Waiting for Connection...</h2>
                                <p className="text-secondary text-sm mt-3">
                                    Once connected, your server or application run will appear here automatically.
                                </p>
                            </>
                        )}
                    </div>
                    <div className="flex-grow overflow-auto">
                        <AppRunList />
                    </div>
                </div>
            </main>

            {/* Status Bar */}
            <StatusBar />
        </>
    );
};
