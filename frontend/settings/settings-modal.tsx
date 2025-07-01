// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { CodeLinkType } from "@/codelink/codelink-model";
import { useAtom, useAtomValue } from "jotai";
import { Moon, Sun } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { AppModel } from "../appmodel";
import { Dropdown } from "../elements/dropdown";
import { Modal } from "../elements/modal";
import { Toggle } from "../elements/toggle";
import { DefaultRpcClient } from "../init";
import { RpcApi } from "../rpc/rpcclientapi";
import { SettingsModel } from "./settings-model";

// Container component that checks isOpen state
export const SettingsModalContainer: React.FC = () => {
    const isOpen = useAtomValue(AppModel.settingsModalOpen);

    // Only render SettingsModal when isOpen is true
    if (!isOpen) return null;

    return <SettingsModal />;
};

// Actual modal component that doesn't need to check isOpen
export const SettingsModal: React.FC = () => {
    const inputRef = useRef<HTMLInputElement>(null);

    // Focus the hidden input when the component mounts
    useEffect(() => {
        // Force blur on any active element first
        if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
        }

        // Then focus our input
        if (inputRef.current) {
            inputRef.current.focus();
        }
    }, []);

    const showSource = useAtomValue(SettingsModel.logsShowSource);
    const showTimestamp = useAtomValue(SettingsModel.logsShowTimestamp);
    const showMilliseconds = useAtomValue(SettingsModel.logsShowMilliseconds);
    const timeFormat = useAtomValue(SettingsModel.logsTimeFormat);
    const showLineNumbers = useAtomValue(SettingsModel.logsShowLineNumbers);
    const emojiReplacement = useAtomValue(SettingsModel.logsEmojiReplacement);
    const codeLinkType = useAtomValue(SettingsModel.codeLinkType);
    const [darkMode, setDarkMode] = useAtom(AppModel.darkMode);

    return (
        <Modal isOpen={true} title="Outrig Settings" onClose={() => AppModel.closeSettingsModal()} className="w-[650px]">
            <div className="text-primary">
                {/* Hidden input to capture focus */}
                <input
                    ref={inputRef}
                    type="text"
                    className="opacity-0 h-0 w-0 absolute"
                    tabIndex={0}
                    aria-hidden="true"
                />

                <div className="p-1 space-y-4">
                    {/* Appearance Section */}
                    <div className="bg-secondary/10 rounded-lg p-4">
                        <h2 className="text-lg font-semibold mb-4 border-b border-secondary/20 pb-2">Appearance</h2>
                        <div className="space-y-4">
                            <Dropdown
                                id="theme-mode"
                                value={darkMode ? "dark" : "light"}
                                onChange={(value) => AppModel.setDarkMode(value === "dark")}
                                options={[
                                    {
                                        value: "light",
                                        label: (
                                            <div className="flex items-center space-x-2">
                                                <Sun size={16} />
                                                <span>Light Mode</span>
                                            </div>
                                        ),
                                    },
                                    {
                                        value: "dark",
                                        label: (
                                            <div className="flex items-center space-x-2">
                                                <Moon size={16} />
                                                <span>Dark Mode</span>
                                            </div>
                                        ),
                                    },
                                ]}
                                label="Theme"
                            />
                        </div>
                    </div>

                    {/* Code Links Section */}
                    <div className="bg-secondary/10 rounded-lg p-4">
                        <h2 className="text-lg font-semibold mb-4 border-b border-secondary/20 pb-2">Code Links</h2>
                        <div className="space-y-4">
                            <Dropdown
                                id="code-link-type"
                                value={codeLinkType || "picker"}
                                onChange={(value) => SettingsModel.setCodeLinkType(value as CodeLinkType)}
                                options={[
                                    { value: "vscode", label: "VSCode" },
                                    { value: "jetbrains", label: "JetBrains IDEs (GoLand)" },
                                    { value: "cursor", label: "Cursor" },
                                    { value: "sublime", label: "Sublime Text" },
                                    { value: "textmate", label: "TextMate" },
                                    { value: "copy", label: "Copy File Path" },
                                    { value: "picker", label: "Show Picker" },
                                ]}
                                label="Link Type"
                            />
                        </div>
                    </div>

                    {/* Logs Section */}
                    <div className="bg-secondary/10 rounded-lg p-4">
                        <h2 className="text-lg font-semibold mb-4 border-b border-secondary/20 pb-2">Logs</h2>
                        <div className="space-y-4">
                            <Toggle
                                id="show-line-numbers"
                                checked={showLineNumbers}
                                onChange={(checked) => SettingsModel.setLogsShowLineNumbers(checked)}
                                label="Show Line Numbers"
                            />

                            <Toggle
                                id="show-source"
                                checked={showSource}
                                onChange={(checked) => SettingsModel.setLogsShowSource(checked)}
                                label="Show Source"
                            />

                            <Toggle
                                id="show-timestamp"
                                checked={showTimestamp}
                                onChange={(checked) => SettingsModel.setLogsShowTimestamp(checked)}
                                label="Show Timestamp"
                            />

                            <Toggle
                                id="show-milliseconds"
                                checked={showMilliseconds}
                                onChange={(checked) => SettingsModel.setLogsShowMilliseconds(checked)}
                                label="Show Milliseconds"
                            />

                            <Dropdown
                                id="time-format"
                                value={timeFormat}
                                onChange={(value) => SettingsModel.setLogsTimeFormat(value as "absolute" | "relative")}
                                options={[
                                    { value: "absolute", label: "Absolute Time" },
                                    { value: "relative", label: "Relative Time" },
                                ]}
                                label="Time Format"
                            />

                            <Dropdown
                                id="emoji-replacement"
                                value={emojiReplacement}
                                onChange={(value) =>
                                    SettingsModel.setLogsEmojiReplacement(value as "never" | "outrig" | "always")
                                }
                                options={[
                                    { value: "never", label: "Never Replace Emojis" },
                                    { value: "outrig", label: "Replace Emojis in Outrig Loggers" },
                                    { value: "always", label: "Always Replace Emojis" },
                                ]}
                                label="Emoji Replacement"
                            />
                        </div>
                    </div>

                    {/* Data Management Section */}
                    <div className="bg-secondary/10 rounded-lg p-4">
                        <h2 className="text-lg font-semibold mb-4 border-b border-secondary/20 pb-2">
                            Data Management
                        </h2>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <div className="font-medium">Clear Non-Active App Runs</div>
                                    <div className="text-sm text-secondary">Remove all disconnected app run data</div>
                                </div>
                                <button
                                    className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white dark:text-black rounded-md cursor-pointer transition-colors"
                                    onClick={async () => {
                                        try {
                                            await RpcApi.ClearNonActiveAppRunsCommand(DefaultRpcClient);
                                            // Trigger a full refresh of the app runs list
                                            AppModel.appRunListModel.triggerFullRefresh();
                                            AppModel.showToast("Success", "Non-active app runs cleared", 3000);
                                        } catch (error) {
                                            console.error("Failed to clear non-active app runs:", error);
                                            AppModel.showToast("Error", "Failed to clear non-active app runs", 3000);
                                        }
                                    }}
                                >
                                    Clear
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </Modal>
    );
};
