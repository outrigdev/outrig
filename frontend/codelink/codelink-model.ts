// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom } from "jotai";

export type CodeLinkType = null | "vscode";

class CodeLinkModel {
    linkTypeAtom = atom<CodeLinkType>("vscode");

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

        return null;
    }
}

export const codeLinkModel = new CodeLinkModel();
