// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React from "react";
import { useAtomValue } from "jotai";
import { cn } from "@/util/util";
import { codeLinkModel } from "./codelink-model";

interface CodeLinkProps {
    file: string;
    children: React.ReactNode;
    className?: string;
}

export const CodeLink: React.FC<CodeLinkProps> = React.memo(({ file, children, className }) => {
    const linkType = useAtomValue(codeLinkModel.linkTypeAtom);
    const { filePath, lineNumber, columnNumber } = codeLinkModel.parseFileString(file);
    const codeLink = codeLinkModel.generateCodeLink(linkType, filePath, lineNumber, columnNumber);

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
});

CodeLink.displayName = "CodeLink";