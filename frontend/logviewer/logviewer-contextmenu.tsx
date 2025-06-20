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

interface LinePtr {
    linenum: number;
    linepage: number;
    lineindex: number;
}

function getSelectedLines(): LinePtr[] {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return [];

    const elements = Array.from(document.querySelectorAll('div[data-linenum]'));

    return elements.filter(el => selection.containsNode(el, true))
        .map(el => ({
            linenum: parseInt(el.getAttribute('data-linenum')!),
            linepage: parseInt(el.getAttribute('data-linepage')!),
            lineindex: parseInt(el.getAttribute('data-lineindex')!)
        }));
}

export function useLogViewerContextMenu(model: LogViewerModel) {
    const { contextMenu, showContextMenu } = useContextMenu();

    const handleContextMenu = useCallback(
        (e: React.MouseEvent, pageNum: number, lineIndex: number) => {
            e.preventDefault();

            // Get selected text and lines
            const selection = window.getSelection();
            const selectedText = selection?.toString() || "";
            const selectedLines = getSelectedLines();

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

            const handleMarkLines = () => {
                if (selectedLines.length > 0) {
                    const lineNumbers = selectedLines.map(line => line.linenum);
                    
                    // Check if all selected lines are already marked
                    const allMarked = lineNumbers.every(lineNum => model.isLineMarked(lineNum));
                    
                    // If all are marked, unmark them; otherwise mark them
                    model.markLines(lineNumbers, !allMarked);
                } else {
                    // No selection, mark just the clicked line
                    const logLine = model.getLogLineByPageAndIndex(pageNum, lineIndex);
                    if (logLine) {
                        model.toggleLineMarked(logLine.linenum);
                    }
                }
            };

            // Determine mark label based on selection and current mark status
            let markLabel: string;
            if (selectedLines.length > 1) {
                const lineNumbers = selectedLines.map(line => line.linenum);
                const allMarked = lineNumbers.every(lineNum => model.isLineMarked(lineNum));
                markLabel = allMarked ? `Unmark ${selectedLines.length} Lines` : `Mark ${selectedLines.length} Lines`;
            } else if (selectedLines.length === 1) {
                const isMarked = model.isLineMarked(selectedLines[0].linenum);
                markLabel = isMarked ? "Unmark Line" : "Mark Line";
            } else {
                // No selection, check the clicked line
                const logLine = model.getLogLineByPageAndIndex(pageNum, lineIndex);
                const isMarked = logLine ? model.isLineMarked(logLine.linenum) : false;
                markLabel = isMarked ? "Unmark Line" : "Mark Line";
            }

            const items = [
                {
                    label: isFullLine ? "Copy Line" : "Copy",
                    onClick: handleCopy,
                    disabled: !text,
                },
                { type: "separator" as const },
                {
                    label: markLabel,
                    onClick: handleMarkLines,
                    disabled: false,
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
