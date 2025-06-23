// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useContextMenu } from "@/elements/usecontextmenu";
import { SettingsModel } from "@/settings/settings-model";
import { getDefaultStore } from "jotai";
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

function nodeToElem(node: Node): Element {
    return node.nodeType === Node.TEXT_NODE ? (node.parentNode! as Element) : (node as Element);
}

function nodeToLinePtr(node: Element): LinePtr | null {
    const linenum = node.getAttribute("data-linenum");
    const linepage = node.getAttribute("data-linepage");
    const lineindex = node.getAttribute("data-lineindex");

    if (!linenum || !linepage || !lineindex) return null;

    return {
        linenum: parseInt(linenum),
        linepage: parseInt(linepage),
        lineindex: parseInt(lineindex),
    };
}

function addLinePtrIfNew(current: Element, linePtrs: LinePtr[], seenLines: Set<number>): void {
    const linePtr = nodeToLinePtr(current);
    if (linePtr && !seenLines.has(linePtr.linenum)) {
        linePtrs.push(linePtr);
        seenLines.add(linePtr.linenum);
    }
}

function makeTreeWalkerForSelection(selection: Selection): TreeWalker {
    const range = selection.getRangeAt(0);
    return document.createTreeWalker(range.commonAncestorContainer, NodeFilter.SHOW_ELEMENT, {
        acceptNode: (node: Element) => {
            if (selection.containsNode(node, false) && node.hasAttribute("data-linenum")) {
                return NodeFilter.FILTER_ACCEPT;
            }
            return NodeFilter.FILTER_SKIP;
        },
    });
}

function getSelectedLines(): LinePtr[] {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return [];

    const range = selection.getRangeAt(0);
    const linePtrs: LinePtr[] = [];
    const seenLines = new Set<number>();

    // FIRST: Check start of selection range and walk up parents
    for (
        let current = nodeToElem(range.startContainer);
        current && current !== document.body;
        current = current.parentElement!
    ) {
        if (!current.hasAttribute || !current.hasAttribute("data-linenum")) continue;
        addLinePtrIfNew(current, linePtrs, seenLines);
        break;
    }

    // SECOND: Check end of selection range
    let endNode = range.endContainer;
    let endElement = nodeToElem(endNode);

    // First check: if the container is user-select none, NO MATCH
    if (getComputedStyle(endElement).userSelect === "none") {
        // End container is user-select:none, skipping
    } else {
        // Container is not user-select:none, we MIGHT have a match
        // But if endOffset is 0 AND previous sibling is user-select:none, then NO MATCH
        let shouldSkip = false;
        if (range.endOffset === 0 && endNode.previousSibling) {
            const prevSibling = endNode.previousSibling;
            const prevElement = nodeToElem(prevSibling);
            if (getComputedStyle(prevElement).userSelect === "none") {
                // EndOffset is 0 and previous sibling is user-select:none, skipping
                shouldSkip = true;
            }
        }

        if (!shouldSkip) {
            // Walk the parents to find a data-linenum node
            for (let current = endElement; current && current !== document.body; current = current.parentElement!) {
                if (!current.hasAttribute) continue;

                if (current.hasAttribute("data-linenum")) {
                    addLinePtrIfNew(current, linePtrs, seenLines);
                    break;
                }
            }
        }
    }

    // THIRD: Walk entire selection and add any element with data-linenum
    const walker = makeTreeWalkerForSelection(selection);

    while (true) {
        const node = walker.nextNode();
        if (!node) break;

        addLinePtrIfNew(node as Element, linePtrs, seenLines);
    }

    // Sort by line number for consistent ordering
    return linePtrs.sort((a, b) => a.linenum - b.linenum);
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
                    const lineNumbers = selectedLines.map((line) => line.linenum);

                    // Check if all selected lines are already marked
                    const allMarked = lineNumbers.every((lineNum) => model.isLineMarked(lineNum));

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
                const lineNumbers = selectedLines.map((line) => line.linenum);
                const allMarked = lineNumbers.every((lineNum) => model.isLineMarked(lineNum));
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

            // Get current settings values
            const store = getDefaultStore();
            const showLineNumbers = store.get(SettingsModel.logsShowLineNumbers);
            const showSource = store.get(SettingsModel.logsShowSource);
            const showTimestamp = store.get(SettingsModel.logsShowTimestamp);

            const handleToggleLineNumbers = () => {
                SettingsModel.setLogsShowLineNumbers(!showLineNumbers);
            };

            const handleToggleSource = () => {
                SettingsModel.setLogsShowSource(!showSource);
            };

            const handleToggleTimestamp = () => {
                SettingsModel.setLogsShowTimestamp(!showTimestamp);
            };

            const handleOpenLogSettings = () => {
                AppModel.openSettingsModal();
            };

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
                { type: "separator" as const },
                {
                    label: showLineNumbers ? "Hide Line Numbers" : "Show Line Numbers",
                    onClick: handleToggleLineNumbers,
                    disabled: false,
                },
                {
                    label: showSource ? "Hide Source" : "Show Source",
                    onClick: handleToggleSource,
                    disabled: false,
                },
                {
                    label: showTimestamp ? "Hide Timestamp" : "Show Timestamp",
                    onClick: handleToggleTimestamp,
                    disabled: false,
                },
                { type: "separator" as const },
                {
                    label: "Open Settings",
                    onClick: handleOpenLogSettings,
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
