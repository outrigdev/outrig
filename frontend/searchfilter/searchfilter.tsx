// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { Tooltip } from "@/elements/tooltip";
import { emitter } from "@/events";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { useAtomValue } from "jotai";
import { Filter } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { DELIMITER_PAIRS, handleDelimiter, handleSelectionWrapping, handleSpecialChar } from "./searchfilter-helpers";

interface SearchFilterProps {
    value: string;
    onValueChange: (value: string) => void;
    placeholder?: string;
    autoFocus?: boolean;
    onOutrigKeyDown?: (keyEvent: OutrigKeyboardEvent) => boolean;
    onEscape?: () => boolean;
    className?: string;
    errorSpans?: SearchErrorSpan[];
}

interface CursorInfo {
    start: number;
    end: number;
}

/**
 * ErrorOverlay component to display red squiggly underlines for error spans
 */
const ErrorOverlay: React.FC<{
    value: string;
    errorSpans: SearchErrorSpan[];
    inputRef: React.RefObject<HTMLInputElement>;
    cursorInfo: CursorInfo;
}> = ({ value, errorSpans, inputRef, cursorInfo }) => {
    if (!errorSpans?.length) {
        return null;
    }

    // Create segments from the error spans
    type Segment = {
        start: number;
        end: number;
        isError: boolean;
        errorMessage?: string;
        isActive?: boolean;
    };

    // Sort error spans by start index to ensure correct segment creation
    const sortedErrorSpans = [...errorSpans].sort((a, b) => a.start - b.start);

    const segments: Segment[] = [];
    let lastIndex = 0;

    // Calculate if cursor is inside a segment
    const isCursorInSegment = (segmentStart: number, segmentEnd: number) => {
        // Only consider cursor positions >= 0 (to handle the -1 case when input is not focused)
        return (
            cursorInfo.start >= 0 &&
            // Cursor is inside the span
            ((cursorInfo.start >= segmentStart && cursorInfo.start < segmentEnd) ||
                // Selection end is inside the span
                (cursorInfo.end > segmentStart && cursorInfo.end <= segmentEnd) ||
                // Selection completely contains the span
                (cursorInfo.start <= segmentStart && cursorInfo.end >= segmentEnd))
        );
    };

    // Create segments for each error span and the text in between
    sortedErrorSpans.forEach((span) => {
        // Add non-error segment before this error span (if any)
        if (span.start > lastIndex) {
            segments.push({
                start: lastIndex,
                end: span.start,
                isError: false,
                isActive: false,
            });
        }

        // Add the error segment
        segments.push({
            start: span.start,
            end: span.end,
            isError: true,
            errorMessage: span.errormessage,
            isActive: isCursorInSegment(span.start, span.end),
        });

        lastIndex = span.end;
    });

    // Add any remaining text after the last error span
    if (lastIndex < value.length) {
        segments.push({
            start: lastIndex,
            end: value.length,
            isError: false,
            isActive: false,
        });
    }

    return (
        <div
            className="absolute inset-0 pointer-events-none font-mono z-10"
            style={{
                top: 0,
                left: 0,
            }}
        >
            <div className="text-transparent whitespace-pre text-sm py-1 pl-0 pr-2 flex flex-row">
                {segments.map((segment, index) => {
                    const text = value.substring(segment.start, segment.end);

                    if (segment.isError) {
                        return (
                            <div key={index} className="relative inline-block">
                                {/* Text content - non-interactive */}
                                <span
                                    className="relative z-10 pointer-events-none underline decoration-wavy decoration-red-500 inline"
                                    style={{ textUnderlineOffset: "2px" }}
                                >
                                    {text}
                                </span>

                                <Tooltip
                                    content={segment.errorMessage || "Error"}
                                    placement="bottom"
                                    forceOpen={segment.isActive}
                                >
                                    <div
                                        className="absolute left-0 w-full z-0 pointer-events-auto"
                                        style={{ bottom: -4, height: 6 }}
                                    />
                                </Tooltip>
                            </div>
                        );
                    }

                    return (
                        <span key={index} className="inline">
                            {text}
                        </span>
                    );
                })}
            </div>
        </div>
    );
};

export const SearchFilter: React.FC<SearchFilterProps> = ({
    value,
    onValueChange,
    placeholder = "Filter...",
    autoFocus = false,
    onOutrigKeyDown,
    onEscape,
    className = "",
    errorSpans = [],
}) => {
    // Track cursor position and selection
    const [cursorInfo, setCursorInfo] = React.useState<CursorInfo>({ start: 0, end: 0 });

    // Get the settings modal state
    const settingsModalOpen = useAtomValue(AppModel.settingsModalOpen);

    // Update cursor position when selection changes
    useEffect(() => {
        const handleSelectionChange = () => {
            // Check if our input is the active element
            if (document.activeElement === inputRef.current) {
                setCursorInfo({
                    start: inputRef.current.selectionStart || 0,
                    end: inputRef.current.selectionEnd || 0,
                });
            } else {
                // If input is not focused, set cursor position to -1 so it won't match any error span
                setCursorInfo({ start: -1, end: -1 });
            }
        };

        // Add event listener to document
        document.addEventListener("selectionchange", handleSelectionChange);

        // Call once on mount to initialize
        handleSelectionChange();

        // Clean up
        return () => {
            document.removeEventListener("selectionchange", handleSelectionChange);
        };
    }, []);
    // Handle keydown events for the search filter
    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        const input = e.currentTarget;
        const key = e.key;

        // Check if text is selected and a delimiter key is pressed
        if (input.selectionStart !== input.selectionEnd && key in DELIMITER_PAIRS) {
            if (handleSelectionWrapping(e, input, key, DELIMITER_PAIRS[key], onValueChange)) {
                return;
            }
        }
        // Only process other delimiter handling if selection is collapsed (no text is selected)
        else if (input.selectionStart === input.selectionEnd) {
            // First check for special character handling
            if (handleSpecialChar(e, input, onValueChange)) {
                return;
            }

            // Then check for delimiter handling
            if (key in DELIMITER_PAIRS) {
                if (handleDelimiter(e, input, key, DELIMITER_PAIRS[key], onValueChange)) {
                    return;
                }
            }

            // Handle closing parenthesis specifically
            if (key === ")") {
                const cursorPos = input.selectionStart;
                if (cursorPos !== null && cursorPos < input.value.length && input.value[cursorPos] === ")") {
                    // If the next character is already the closing parenthesis, just move the cursor past it
                    e.preventDefault();
                    input.setSelectionRange(cursorPos + 1, cursorPos + 1);
                    return;
                }
            }
        }

        // If we didn't handle the key, pass to the regular handler
        keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
            // Handle Escape key
            if (checkKeyPressed(keyEvent, "Escape")) {
                // First check if parent wants to handle Escape
                if (onEscape && onEscape()) {
                    // Parent handled it, don't clear search
                    return true;
                }
                
                // Parent didn't handle it or no handler provided, clear search
                onValueChange("");
                return true;
            }

            // Pass other keys to the provided handler
            if (onOutrigKeyDown) {
                return onOutrigKeyDown(keyEvent);
            }

            return false;
        })(e);
    };
    // Create internal ref if no external ref is provided
    const inputRef = useRef<HTMLInputElement>(null);

    // Also update cursor position when input value changes
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        onValueChange(e.target.value);
        // We need to wait for React to update the input value before getting cursor position
        setTimeout(() => {
            if (inputRef.current) {
                setCursorInfo({
                    start: inputRef.current.selectionStart || 0,
                    end: inputRef.current.selectionEnd || 0,
                });
            }
        }, 0);
    };

    // Handle focus management
    useEffect(() => {
        if (!autoFocus) return;

        // Focus on mount, but only if settings modal is not open
        const timer = setTimeout(() => {
            if (!settingsModalOpen) {
                inputRef.current?.focus();
            }
        }, 50);

        // Handle window focus changes
        const handleWindowFocus = () => {
            // Only focus if the settings modal is not open
            if (autoFocus && !settingsModalOpen) {
                inputRef.current?.focus();
            }
        };

        window.addEventListener("focus", handleWindowFocus);

        return () => {
            clearTimeout(timer);
            window.removeEventListener("focus", handleWindowFocus);
        };
    }, [autoFocus, inputRef, settingsModalOpen]);

    // Listen for modalclose and focussearch events to focus the input
    useEffect(() => {
        const handleModalClose = () => {
            if (autoFocus) {
                inputRef.current?.focus();
            }
        };

        const handleFocusSearch = () => {
            inputRef.current?.focus();
        };

        // Add event listeners
        emitter.on("modalclose", handleModalClose);
        emitter.on("focussearch", handleFocusSearch);

        // Clean up
        return () => {
            emitter.off("modalclose", handleModalClose);
            emitter.off("focussearch", handleFocusSearch);
        };
    }, [autoFocus, inputRef]);

    return (
        <div className={`flex items-center flex-grow ${className}`}>
            <div className="select-none pr-2 text-muted w-10 text-right font-mono flex justify-end items-center">
                <Filter size={16} className="text-accent" fill="currentColor" stroke="currentColor" strokeWidth={1} />
            </div>
            <div className="relative w-full">
                <input
                    ref={inputRef}
                    type="text"
                    placeholder={placeholder}
                    value={value}
                    onChange={handleChange}
                    onKeyDown={handleKeyDown}
                    spellCheck="false"
                    autoCorrect="off"
                    autoCapitalize="off"
                    autoComplete="off"
                    className="w-full bg-transparent text-primary translate-y-px placeholder:text-secondary text-sm py-1 pl-0 pr-2 
                      border-none ring-0 outline-none focus:outline-none focus:ring-0 font-mono"
                />
                {errorSpans?.length > 0 && (
                    <ErrorOverlay
                        key={`error-overlay-${value}`} // Force re-mount when value changes
                        value={value}
                        errorSpans={errorSpans}
                        inputRef={inputRef}
                        cursorInfo={cursorInfo}
                    />
                )}
            </div>
        </div>
    );
};
