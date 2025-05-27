// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React from "react";
import { cn } from "@/util/util";
import { generateCodeLink, parseFileString } from "./codelink-model";

interface CodeLinkProps {
    file: string;
    children: React.ReactNode;
    className?: string;
}

export const CodeLink: React.FC<CodeLinkProps> = ({ file, children, className }) => {
    const { filePath, lineNumber, columnNumber } = parseFileString(file);
    const codeLink = generateCodeLink(filePath, lineNumber, columnNumber, "vscode");

    return codeLink ? (
        <a
            href={codeLink.href}
            onClick={codeLink.onClick}
            className={cn("cursor-pointer hover:text-blue-500 dark:hover:text-blue-300 hover:bg-transparent", className)}
        >
            {children}
        </a>
    ) : (
        <span className={className}>
            {children}
        </span>
    );
};