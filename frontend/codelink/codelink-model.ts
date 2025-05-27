// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Type for editor link options
export type CodeLinkType = null | "vscode";

// Parse a file string that can be in formats:
// - "filename.go"
// - "filename.go:123"
// - "filename.go:123:45"
export function parseFileString(fileStr: string): { filePath: string; lineNumber?: number; columnNumber?: number } {
    const parts = fileStr.split(':');
    
    if (parts.length === 1) {
        return { filePath: parts[0] };
    } else if (parts.length === 2) {
        return {
            filePath: parts[0],
            lineNumber: parseInt(parts[1], 10) || undefined
        };
    } else if (parts.length >= 3) {
        return {
            filePath: parts[0],
            lineNumber: parseInt(parts[1], 10) || undefined,
            columnNumber: parseInt(parts[2], 10) || undefined
        };
    }
    
    return { filePath: fileStr };
}

// Generate a code link for a file path and line number
export function generateCodeLink(
    filePath: string,
    lineNumber?: number,
    columnNumber?: number,
    linkType: CodeLinkType = "vscode"
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