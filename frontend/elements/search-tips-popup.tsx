// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { arrow, autoUpdate, flip, FloatingArrow, FloatingPortal, offset, shift, useFloating } from "@floating-ui/react";
import { Lightbulb, X } from "lucide-react";
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
                    className="absolute top-2 right-2 text-muted hover:text-primary dark:text-gray-400 dark:hover:text-white cursor-pointer"
                    onClick={onClose}
                    aria-label="Close search tips"
                >
                    <X size={16} />
                </button>

                {/* Content */}
                <div className="p-2 max-w-[500px]">
                    <div className="flex items-center gap-2 mb-3">
                        <Lightbulb size={18} className="text-amber-500" />
                        <div className="font-semibold">Search Cheat Sheet</div>
                    </div>

                    {/* Main grid layout */}
                    <div className="grid grid-cols-2 gap-2 text-xs">
                        {/* Left column */}
                        <div className="space-y-0.5">
                            <div className="rounded-md p-1">
                                <div className="font-medium text-purple-600 dark:text-purple-400 flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-purple-500 rounded-full"></span> Case
                                    Sensitive
                                </div>
                                <div className="space-y-0.5">
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-purple-700 dark:text-purple-300">
                                            'Error:'
                                        </code>
                                        <span className="text-[10px]">Match Case</span>
                                    </div>
                                </div>
                            </div>

                            <div className="rounded-md p-1">
                                <div className="font-medium text-green-600 dark:text-green-400 flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-green-500 rounded-full"></span> Regex
                                </div>
                                <div className="space-y-0.5">
                                    <div className="flex justify-between gap-1 items-end">
                                        <code className="font-mono px-1 rounded text-green-700 dark:text-green-200">
                                            /^error:.*db/
                                        </code>
                                        <span className="text-[10px]">Ignore Case</span>
                                    </div>
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-green-700 dark:text-green-200">
                                            c/^Error:/
                                        </code>
                                        <span className="text-[10px]">Match Case</span>
                                    </div>
                                </div>
                            </div>

                            <div className="rounded-md p-1">
                                <div className="font-medium text-blue-600 dark:text-blue-400 flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-blue-500 rounded-full"></span> Fuzzy
                                </div>
                                <div className="space-y-0.5">
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-blue-700 dark:text-blue-200">
                                            ~dbconnerr
                                        </code>
                                        <span className="text-[10px]">Fuzzy Search</span>
                                    </div>
                                </div>
                            </div>
                        </div>

                        {/* Right column */}
                        <div className="space-y-0.5">
                            {/* Combining */}
                            <div className="rounded-md p-1">
                                <div className="font-medium text-amber-600 dark:text-amber-400 flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-amber-500 rounded-full"></span> Combining
                                </div>
                                <div className="space-y-0.5">
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-amber-700 dark:text-amber-100">
                                            timeout db
                                        </code>
                                        <span className="text-[10px]">AND</span>
                                    </div>
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-amber-700 dark:text-amber-100">
                                            timeout | retry
                                        </code>
                                        <span className="text-[10px]">OR</span>
                                    </div>
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-amber-700 dark:text-amber-100">
                                            -timeout
                                        </code>
                                        <span className="text-[10px]">NOT</span>
                                    </div>
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded text-amber-700 dark:text-amber-100">
                                            db (t1 | t2)
                                        </code>
                                        <span className="text-[10px]">Group</span>
                                    </div>
                                </div>
                            </div>

                            {/* Advanced */}
                            <div className="rounded-md p-1">
                                <div className="font-medium text-red-600 dark:text-red-400 flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-red-500 rounded-full"></span> Advanced
                                </div>
                                <div className="space-y-0.5">
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded dark:text-red-100 text-red-900">
                                            $state:"io wait"
                                        </code>
                                        <span className="text-[10px]">Fields</span>
                                    </div>
                                    <div className="flex justify-between items-end">
                                        <code className="font-mono px-1 rounded dark:text-red-100 text-red-900">
                                            #backend
                                        </code>
                                        <span className="text-[10px]">Tags</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        {/* Examples section - more compact */}
                        <div className="col-span-2 space-y-0.5">
                            <div className="rounded-md p-1">
                                <div className="font-medium text-primary flex items-center gap-1">
                                    <span className="inline-block w-2 h-2 bg-primary rounded-full"></span> Examples
                                </div>
                                <div className="px-2 py-1 bg-panel rounded-md font-mono text-[12px] overflow-x-auto">
                                    <div className="text-emerald-500"># HTTP errors excluding 404s</div>
                                    <div>(http | https) error -/404\s+Not\s+Found/</div>
                                    <div className="mt-1 text-emerald-500"># Database errors in backend</div>
                                    <div>(db | database) #backend (error | failure)</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="flex justify-between items-center mt-2">
                        <a
                            href="https://outrig.run/docs/search"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-primary hover:underline cursor-pointer"
                        >
                            Full Search Documentation →
                        </a>
                        <button
                            onClick={onClose}
                            className="text-xs text-primary bg-primary/20 px-3 py-1 rounded hover:bg-primary/30 cursor-pointer transition-colors"
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
