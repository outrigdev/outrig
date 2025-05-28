// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";

export type CodeLinkType = null | "vscode" | "jetbrains" | "cursor" | "sublime" | "textmate" | "copy" | "picker";

class CodeLinkModel {
    pendingCodeLink: PrimitiveAtom<{filePath: string; lineNumber?: number; columnNumber?: number} | null> = atom(null) as PrimitiveAtom<{filePath: string; lineNumber?: number; columnNumber?: number} | null>;

    parseFileString(fileStr: string): { filePath: string; lineNumber?: number; columnNumber?: number } {
        const parts = fileStr.split(":");

        if (parts.length === 1) {
            return { filePath: parts[0] };
        } else if (parts.length === 2) {
            return {
                filePath: parts[0],
                lineNumber: parseInt(parts[1], 10) || undefined,
            };
        } else if (parts.length >= 3) {
            return {
                filePath: parts[0],
                lineNumber: parseInt(parts[1], 10) || undefined,
                columnNumber: parseInt(parts[2], 10) || undefined,
            };
        }

        return { filePath: fileStr };
    }

    generateCodeLink(
        linkType: CodeLinkType,
        filePath: string,
        lineNumber?: number,
        columnNumber?: number
    ): { href: string; onClick: () => null } | null {
        if (linkType == null) {
            return null;
        }

        if (linkType === "vscode") {
            let href = `vscode://file${filePath}`;

            if (lineNumber) {
                href += `:${lineNumber}`;
                if (columnNumber) {
                    href += `:${columnNumber}`;
                }
            }

            return {
                href,
                onClick: () => null,
            };
        }

        if (linkType === "jetbrains") {
            let href = `jetbrains://open?file=${encodeURIComponent(filePath)}`;

            if (lineNumber) {
                href += `&line=${lineNumber}`;
                if (columnNumber) {
                    href += `&column=${columnNumber}`;
                }
            }

            return {
                href,
                onClick: () => null,
            };
        }

        if (linkType === "cursor") {
            let href = `cursor://file${filePath}`;

            if (lineNumber) {
                href += `:${lineNumber}`;
                if (columnNumber) {
                    href += `:${columnNumber}`;
                }
            }

            return {
                href,
                onClick: () => null,
            };
        }

        if (linkType === "sublime") {
            let href = `subl://open?url=file://${encodeURIComponent(filePath)}`;

            if (lineNumber) {
                href += `&line=${lineNumber}`;
                if (columnNumber) {
                    href += `&column=${columnNumber}`;
                }
            }

            return {
                href,
                onClick: () => null,
            };
        }

        if (linkType === "textmate") {
            let href = `txmt://open?url=file://${encodeURIComponent(filePath)}`;

            if (lineNumber) {
                href += `&line=${lineNumber}`;
                if (columnNumber) {
                    href += `&column=${columnNumber}`;
                }
            }

            return {
                href,
                onClick: () => null,
            };
        }

        if (linkType === "copy") {
            return {
                href: "#",
                onClick: () => {
                    navigator.clipboard.writeText(filePath);
                    return null;
                },
            };
        }

        if (linkType === "picker") {
            return {
                href: "#",
                onClick: () => {
                    // Store the pending link information
                    getDefaultStore().set(this.pendingCodeLink, {
                        filePath,
                        lineNumber,
                        columnNumber,
                    });
                    AppModel.openCodeLinkPickerModal();
                    return null;
                },
            };
        }

        return null;
    }
}

export const codeLinkModel = new CodeLinkModel();
