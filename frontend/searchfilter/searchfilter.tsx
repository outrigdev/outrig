import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { Filter } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { DELIMITER_PAIRS, handleDelimiter, handleSelectionWrapping, handleSpecialChar } from "./searchfilter-helpers";

interface SearchFilterProps {
    value: string;
    onValueChange: (value: string) => void;
    placeholder?: string;
    autoFocus?: boolean;
    onOutrigKeyDown?: (keyEvent: OutrigKeyboardEvent) => boolean;
    className?: string;
    errorSpans?: SearchErrorSpan[];
}

/**
 * ErrorOverlay component to display red squiggly underlines for error spans
 */
const ErrorOverlay: React.FC<{
    value: string;
    errorSpans: SearchErrorSpan[];
    inputRef: React.RefObject<HTMLInputElement>;
}> = ({ value, errorSpans, inputRef }) => {
    if (!errorSpans?.length) {
        return null;
    }

    // Create segments from the error spans
    type Segment = {
        start: number;
        end: number;
        isError: boolean;
        errorMessage?: string;
    };

    // Sort error spans by start index to ensure correct segment creation
    const sortedErrorSpans = [...errorSpans].sort((a, b) => a.start - b.start);

    const segments: Segment[] = [];
    let lastIndex = 0;

    // Create segments for each error span and the text in between
    sortedErrorSpans.forEach((span) => {
        // Add non-error segment before this error span (if any)
        if (span.start > lastIndex) {
            segments.push({
                start: lastIndex,
                end: span.start,
                isError: false,
            });
        }

        // Add the error segment
        segments.push({
            start: span.start,
            end: span.end,
            isError: true,
            errorMessage: span.errormessage,
        });

        lastIndex = span.end;
    });

    // Add any remaining text after the last error span
    if (lastIndex < value.length) {
        segments.push({
            start: lastIndex,
            end: value.length,
            isError: false,
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
                            <span
                                key={index}
                                className="underline decoration-wavy decoration-red-500 inline"
                                style={{ textUnderlineOffset: "2px" }}
                            >
                                {text}
                            </span>
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
    className = "",
    errorSpans = [],
}) => {
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
        }

        // If we didn't handle the key, pass to the regular handler
        keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
            // Handle Escape key internally
            if (checkKeyPressed(keyEvent, "Escape")) {
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

    // Handle focus management
    useEffect(() => {
        if (!autoFocus) return;

        // Focus on mount
        const timer = setTimeout(() => {
            inputRef.current?.focus();
        }, 50);

        // Handle tab/window visibility changes
        const handleVisibilityChange = () => {
            if (!document.hidden && autoFocus) {
                inputRef.current?.focus();
            }
        };

        document.addEventListener("visibilitychange", handleVisibilityChange);

        return () => {
            clearTimeout(timer);
            document.removeEventListener("visibilitychange", handleVisibilityChange);
        };
    }, [autoFocus, inputRef]);

    return (
        <div className={`flex items-center flex-grow ${className}`}>
            <div className="select-none pr-2 text-muted w-10 text-right font-mono flex justify-end items-center">
                <Filter size={16} className="text-muted" fill="currentColor" stroke="currentColor" strokeWidth={1} />
            </div>
            <div className="relative w-full">
                <input
                    ref={inputRef}
                    type="text"
                    placeholder={placeholder}
                    value={value}
                    onChange={(e) => onValueChange(e.target.value)}
                    onKeyDown={handleKeyDown}
                    className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2 
                      border-none ring-0 outline-none focus:outline-none focus:ring-0 font-mono"
                />
                {errorSpans?.length > 0 && (
                    <ErrorOverlay
                        key={`error-overlay-${value}`} // Force re-mount when value changes
                        value={value}
                        errorSpans={errorSpans}
                        inputRef={inputRef}
                    />
                )}
            </div>
        </div>
    );
};
