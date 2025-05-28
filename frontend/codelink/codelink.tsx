// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import { useAtomValue } from "jotai";
import React from "react";
import { codeLinkModel } from "./codelink-model";

interface CodeLinkProps {
    file: string;
    children: React.ReactNode;
    className?: string;
}

export const CodeLink: React.FC<CodeLinkProps> = React.memo(({ file, children, className }) => {
    const linkType = useAtomValue(codeLinkModel.linkTypeAtom);
    const parsedFile = codeLinkModel.parseFileString(file);

    if (!parsedFile) {
        return <span className={className}>{children}</span>;
    }

    const { filePath, lineNumber, columnNumber } = parsedFile;
    const codeLink = codeLinkModel.generateCodeLink(linkType, filePath, lineNumber, columnNumber);

    return codeLink ? (
        <a
            href={codeLink.href}
            onClick={codeLink.onClick}
            className={cn(
                "cursor-pointer hover:text-blue-500 dark:hover:text-blue-300 hover:bg-transparent",
                className
            )}
        >
            {children}
        </a>
    ) : (
        <span className={className}>{children}</span>
    );
});

CodeLink.displayName = "CodeLink";
