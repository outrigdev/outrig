// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Modal } from "@/elements/modal";
import { SettingsModel } from "@/settings/settings-model";
import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import { Code, Copy, FileText, Terminal } from "lucide-react";
import React from "react";
import { AppModel } from "../appmodel";
import { CodeLinkType, codeLinkModel } from "./codelink-model";

const editorOptions: Array<{
    type: CodeLinkType;
    name: string;
    description: string;
    icon: React.ReactNode;
}> = [
    {
        type: "vscode",
        name: "VS Code",
        description: "Open files in Visual Studio Code",
        icon: <Code size={20} />,
    },
    {
        type: "cursor",
        name: "Cursor",
        description: "Open files in Cursor editor",
        icon: <Terminal size={20} />,
    },
    {
        type: "jetbrains",
        name: "JetBrains IDEs",
        description: "Open files in IntelliJ, GoLand, WebStorm, etc.",
        icon: <Code size={20} />,
    },
    {
        type: "sublime",
        name: "Sublime Text",
        description: "Open files in Sublime Text",
        icon: <FileText size={20} />,
    },
    {
        type: "textmate",
        name: "TextMate",
        description: "Open files in TextMate",
        icon: <FileText size={20} />,
    },
    {
        type: "copy",
        name: "Copy Path",
        description: "Copy file path to clipboard",
        icon: <Copy size={20} />,
    },
];

export function CodeLinkPickerModalContainer() {
    const isOpen = useAtomValue(AppModel.codeLinkPickerModalOpen);
    const pendingLink = useAtomValue(codeLinkModel.pendingCodeLink);

    const handleClose = () => {
        AppModel.closeCodeLinkPickerModal();
    };

    const handleEditorSelect = (linkType: CodeLinkType) => {
        // Save the setting
        SettingsModel.setCodeLinkType(linkType);
        
        // Perform the actual deep link if we have pending link info
        if (pendingLink) {
            const linkResult = codeLinkModel.generateCodeLink(
                linkType,
                pendingLink.filePath,
                pendingLink.lineNumber,
                pendingLink.columnNumber
            );
            
            if (linkResult) {
                if (linkResult.href !== "#") {
                    // For URL-based links, open them
                    window.open(linkResult.href, "_self");
                } else {
                    // For onClick-based links (like copy), execute the onClick
                    linkResult.onClick();
                }
            }
        }
        
        AppModel.closeCodeLinkPickerModal();
    };

    return (
        <Modal isOpen={isOpen} title="Choose Your Editor" onClose={handleClose} className="w-[600px]">
            <div className="space-y-4">
                <p className="text-muted text-sm">
                    Outrig can deep link into your favorite editor. Please select the editor you use to open code files:
                </p>
                
                <div className="grid gap-3">
                    {editorOptions.map((option) => (
                        <button
                            key={option.type}
                            onClick={() => handleEditorSelect(option.type)}
                            className={cn(
                                "flex items-center gap-3 p-3 rounded-md border border-border",
                                "hover:bg-hover hover:border-accent cursor-pointer",
                                "transition-colors duration-150",
                                "text-left w-full"
                            )}
                        >
                            <div className="text-accent">{option.icon}</div>
                            <div className="flex-1">
                                <div className="font-medium text-primary">{option.name}</div>
                                <div className="text-sm text-muted">{option.description}</div>
                            </div>
                        </button>
                    ))}
                </div>
                
                <p className="text-xs text-muted mt-4">
                    You can change this setting later in the Settings menu.
                </p>
            </div>
        </Modal>
    );
}