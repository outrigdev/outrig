// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { useContextMenu } from "@/elements/usecontextmenu";
import React, { useCallback } from "react";
import { LogViewerModel } from "./logviewer-model";

async function copySelectionText(): Promise<string> {
    const selection = window.getSelection();
    if (!selection) return "";

    const range = selection.getRangeAt(0);
    const clonedSelection = range.cloneContents();

    const div = document.createElement("div");
    div.style.position = "absolute";
    div.style.left = "-99999px";
    div.appendChild(clonedSelection);

    // Remove elements with user-select:none
    div.querySelectorAll("*").forEach((el) => {
        if (getComputedStyle(el).userSelect === "none") {
            el.remove();
        }
    });

    const textToCopy = (div.textContent || "").replace(/\n+$/, "");
    await navigator.clipboard.writeText(textToCopy);
    return textToCopy;
}

export function useLogViewerContextMenu(model: LogViewerModel) {
    const { contextMenu, showContextMenu } = useContextMenu();

    const handleContextMenu = useCallback(
        (e: React.MouseEvent, pageNum: number, lineIndex: number) => {
            e.preventDefault();

            // Get selected text
            const selection = window.getSelection();
            const selectedText = selection?.toString() || "";

            let text: string;
            let isFullLine: boolean;

            if (selectedText.trim()) {
                // User has selected text
                text = selectedText;
                isFullLine = false;
            } else {
                // No selection, get the full line content using page and index
                const logLine = model.getLogLineByPageAndIndex(pageNum, lineIndex);
                if (!logLine) return;
                text = logLine.msg;
                isFullLine = true;
            }

            const handleCopy = async () => {
                if (isFullLine) {
                    // For full line copy, use the message content directly
                    const textToCopy = text.replace(/\n+$/, "");
                    await navigator.clipboard.writeText(textToCopy);
                } else {
                    // For selected text, use the improved selection handling
                    await copySelectionText();
                }
            };

            const items = [
                {
                    label: isFullLine ? "Copy Line" : "Copy",
                    onClick: handleCopy,
                    disabled: !text,
                },
            ];

            showContextMenu(e, items);
        },
        [model, showContextMenu]
    );

    return {
        contextMenu,
        handleContextMenu,
    };
}
