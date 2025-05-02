// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { arrow, autoUpdate, flip, FloatingArrow, FloatingPortal, offset, shift, useFloating } from "@floating-ui/react";
import { X } from "lucide-react";
import React, { useRef } from "react";

interface SearchTipsPopupProps {
    referenceElement: HTMLElement | null;
    isOpen: boolean;
    onClose: () => void;
}

export const SearchTipsPopup: React.FC<SearchTipsPopupProps> = ({ referenceElement, isOpen, onClose }) => {
    const arrowRef = useRef<SVGSVGElement>(null);

    const { refs, floatingStyles, context } = useFloating({
        elements: {
            reference: referenceElement,
        },
        open: isOpen,
        placement: "bottom",
        middleware: [
            offset(15),
            flip(),
            shift({ padding: 12 }),
            arrow({
                element: arrowRef,
                padding: 8,
            }),
        ],
        whileElementsMounted: autoUpdate,
    });

    if (!isOpen) {
        return null;
    }

    return (
        <FloatingPortal>
            <div
                ref={refs.setFloating}
                style={floatingStyles}
                className="bg-white text-primary rounded-lg shadow-xl border-2 border-secondary z-50"
            >
                {/* Close button */}
                <button
                    className="absolute top-2 right-2 text-muted hover:text-primary cursor-pointer"
                    onClick={onClose}
                    aria-label="Close search tips"
                >
                    <X size={16} />
                </button>

                {/* Content */}
                <div className="p-4 max-w-md">
                    <div className="font-semibold mb-2">Search Tips</div>
                    <ul className="text-xs space-y-1 mb-3">
                        <li>
                            <code>term</code> - Find logs containing term
                        </li>
                        <li>
                            <code>'Term'</code> - Case-sensitive search
                        </li>
                        <li>
                            <code>/regex/</code> - Regular expression search
                        </li>
                        <li>
                            <code>term1 term2</code> - AND (both terms)
                        </li>
                        <li>
                            <code>term1 | term2</code> - OR (either term)
                        </li>
                        <li>
                            <code>-term</code> - Exclude term
                        </li>
                        <li>
                            <code>~term</code> - Fuzzy search
                        </li>
                        <li>
                            <code>(term1 | term2) term3</code> - Group expressions
                        </li>
                        <li>
                            <code>$field:value</code> - Field-specific search
                        </li>
                        <li>
                            <code>#tag</code> - Tag search
                        </li>
                    </ul>
                    <div className="text-xs font-medium mb-2">Examples:</div>
                    <pre className="text-xs bg-black/10 p-2 rounded mb-3 overflow-x-auto font-mono whitespace-pre-wrap">
                        {`# Find logs with both "error" and "timeout"
error timeout

# Find logs with either "error" or "warning"
error | warning

# Find logs starting with "error:" and containing "db"
/^error:.*db/

# Find "error" but exclude "retried successfully"
error -"retried successfully"`}
                    </pre>
                    <div className="flex justify-between items-center">
                        <a
                            href="https://outrig.run/docs/search"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-primary hover:underline"
                        >
                            View full search documentation →
                        </a>
                        <button
                            onClick={onClose}
                            className="text-xs bg-primary text-white px-3 py-1 rounded hover:bg-primary/90 cursor-pointer"
                        >
                            Close
                        </button>
                    </div>
                </div>

                {/* Arrow */}
                <FloatingArrow
                    ref={arrowRef}
                    context={context}
                    fill="var(--color-white)"
                    stroke="var(--color-secondary)"
                    strokeWidth={2}
                    height={8}
                    width={16}
                    tipRadius={0}
                />
            </div>
        </FloatingPortal>
    );
};
