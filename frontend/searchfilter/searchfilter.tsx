// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { Tooltip } from "@/elements/tooltip";
import { emitter } from "@/events";
import { SearchStore } from "@/store/searchstore";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { cn } from "@/util/util";
import { getDefaultStore, useAtomValue } from "jotai";
import { Filter, X } from "lucide-react";
import React, { useEffect, useImperativeHandle, useRef, useState } from "react";
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

/**
 * SearchHistoryDropdown component to display and manage search history
 */
interface SearchHistoryDropdownProps {
    onClose: () => void;
    onSelect: (value: string) => void;
    inputRef: React.RefObject<HTMLInputElement>;
}

interface SearchHistoryDropdownHandle {
    handleKeyDown: (keyEvent: OutrigKeyboardEvent) => boolean;
}

const SearchHistoryDropdown = React.forwardRef<SearchHistoryDropdownHandle, SearchHistoryDropdownProps>(
    ({ onClose, onSelect, inputRef }, ref) => {
        const [searchHistory, setSearchHistory] = useState<string[]>([]);
        const [selectedIndex, setSelectedIndex] = useState(-1);
        const historyDropdownRef = useRef<HTMLDivElement>(null);

        // Get the current app run info for search history
        const appRunId = useAtomValue(AppModel.selectedAppRunId);
        const tabName = useAtomValue(AppModel.selectedTab);

        // Load search history when app run or tab changes
        useEffect(() => {
            if (!appRunId) return;

            const store = getDefaultStore();
            const appRunInfoAtom = AppModel.getAppRunInfoAtom(appRunId);
            const appRunInfo = store.get(appRunInfoAtom);

            if (appRunInfo) {
                const history = SearchStore.getSearchHistory(appRunInfo.appname, appRunId, tabName);
                setSearchHistory(history);
            }
        }, [appRunId, tabName]);

        // Reset selected index when dropdown mounts
        useEffect(() => {
            if (searchHistory.length > 0) {
                setSelectedIndex(0);
            } else {
                setSelectedIndex(-1);
            }
        }, [searchHistory.length]);

        // Close history dropdown when clicking outside
        useEffect(() => {
            const handleClickOutside = (event: MouseEvent) => {
                if (
                    historyDropdownRef.current &&
                    !historyDropdownRef.current.contains(event.target as Node) &&
                    inputRef.current &&
                    !inputRef.current.contains(event.target as Node)
                ) {
                    onClose();
                }
            };

            document.addEventListener("mousedown", handleClickOutside);
            return () => {
                document.removeEventListener("mousedown", handleClickOutside);
            };
        }, [onClose, inputRef]);

        // Handle removing a search history item
        const handleRemoveHistoryItem = (e: React.MouseEvent, index: number) => {
            e.stopPropagation();

            if (appRunId) {
                const store = getDefaultStore();
                const appRunInfoAtom = AppModel.getAppRunInfoAtom(appRunId);
                const appRunInfo = store.get(appRunInfoAtom);

                if (appRunInfo) {
                    const termToRemove = searchHistory[index];
                    SearchStore.removeFromSearchHistory(appRunInfo.appname, appRunId, tabName, termToRemove);

                    // Update local history state
                    const updatedHistory = SearchStore.getSearchHistory(appRunInfo.appname, appRunId, tabName);
                    setSearchHistory(updatedHistory);

                    // Adjust selected index if needed
                    if (selectedIndex >= updatedHistory.length) {
                        setSelectedIndex(updatedHistory.length - 1);
                    }
                }
            }
        };

        // Expose methods to parent via ref
        useImperativeHandle(ref, () => ({
            handleKeyDown: (keyEvent: OutrigKeyboardEvent) => {
                if (checkKeyPressed(keyEvent, "ArrowDown")) {
                    if (searchHistory.length > 0) {
                        setSelectedIndex((prev) => (prev >= searchHistory.length - 1 ? 0 : prev + 1));
                    }
                    return true;
                }
                if (checkKeyPressed(keyEvent, "ArrowUp")) {
                    if (searchHistory.length > 0) {
                        setSelectedIndex((prev) => (prev <= 0 ? searchHistory.length - 1 : prev - 1));
                    }
                    return true;
                }
                if (checkKeyPressed(keyEvent, "Enter")) {
                    if (selectedIndex >= 0 && selectedIndex < searchHistory.length) {
                        onSelect(searchHistory[selectedIndex]);
                        onClose();
                    }
                    return true;
                }
                if (checkKeyPressed(keyEvent, "Escape")) {
                    onClose();
                    return true;
                }
                return false;
            },
        }));

        return (
            <div
                ref={historyDropdownRef}
                className="absolute z-50 w-full bg-panel mt-1 border border-strongborder rounded-md shadow-lg shadow-shadow max-h-60 overflow-auto"
            >
                {searchHistory.length > 0 ? (
                    <ul>
                        {searchHistory.map((historyItem, index) => (
                            <li
                                key={`${historyItem}-${index}`}
                                className={cn(
                                    "px-3 py-2 flex justify-between items-center cursor-pointer text-sm font-mono group",
                                    index === selectedIndex
                                        ? "bg-accentbg/20 text-accent"
                                        : "text-primary hover:bg-buttonhover"
                                )}
                                onClick={() => {
                                    onSelect(historyItem);
                                    onClose();
                                }}
                            >
                                <span className="truncate">{historyItem}</span>
                                <Tooltip content="Remove Search from History" placement="top">
                                    <button
                                        className="ml-2 p-1 rounded-full hover:bg-buttonbg text-muted hover:text-primary cursor-pointer opacity-0 group-hover:opacity-100 transition-opacity"
                                        onClick={(e) => handleRemoveHistoryItem(e, index)}
                                        aria-label="Remove from history"
                                    >
                                        <X size={14} />
                                    </button>
                                </Tooltip>
                            </li>
                        ))}
                    </ul>
                ) : (
                    <div className="px-3 py-4 text-center">
                        <p className="text-primary font-medium text-sm">No Search History</p>
                        <p className="text-secondary text-sm">To save a search to history press Enter &crarr;</p>
                    </div>
                )}
            </div>
        );
    }
);

// Add display name for React DevTools
SearchHistoryDropdown.displayName = "SearchHistoryDropdown";

export const SearchFilter: React.FC<SearchFilterProps> = ({
    value,
    onValueChange,
    placeholder = "Filter...",
    autoFocus = false,
    onOutrigKeyDown,
    className = "",
    errorSpans = [],
}) => {
    // Track cursor position and selection
    const [cursorInfo, setCursorInfo] = React.useState<CursorInfo>({ start: 0, end: 0 });

    // State for search history dropdown
    const [isHistoryOpen, setIsHistoryOpen] = useState(false);

    // Create refs for the input element and dropdown
    const inputRef = useRef<HTMLInputElement>(null);
    const dropdownRef = useRef<SearchHistoryDropdownHandle>(null);

    // Get the settings modal state
    const settingsModalOpen = useAtomValue(AppModel.settingsModalOpen);

    // Get the current app run info for search history
    const appRunId = useAtomValue(AppModel.selectedAppRunId);
    const tabName = useAtomValue(AppModel.selectedTab);

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

        // Handle all other key events
        keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
            // If history dropdown is open, delegate keyboard events to it
            if (isHistoryOpen && dropdownRef.current) {
                const handled = dropdownRef.current.handleKeyDown(keyEvent);
                if (handled) {
                    return true;
                }
            }

            // Open history dropdown on arrow down/up when it's not open
            if ((checkKeyPressed(keyEvent, "ArrowUp") || checkKeyPressed(keyEvent, "ArrowDown")) && !isHistoryOpen) {
                setIsHistoryOpen(true);
                return true;
            }

            // Handle Enter key to save search history
            if (checkKeyPressed(keyEvent, "Enter")) {
                const store = getDefaultStore();
                const appRunId = store.get(AppModel.selectedAppRunId);
                const tabName = store.get(AppModel.selectedTab);

                if (appRunId) {
                    // Get the app info to get the app name
                    const appRunInfoAtom = AppModel.getAppRunInfoAtom(appRunId);
                    const appRunInfo = store.get(appRunInfoAtom);

                    if (appRunInfo) {
                        // Save the search term to history
                        SearchStore.saveSearchHistory(appRunInfo.appname, appRunId, tabName);
                    }
                }

                // Don't return true here so that other handlers can still process the Enter key
            }

            // Handle Escape key
            if (checkKeyPressed(keyEvent, "Escape")) {
                // First check if search tips are open
                const isSearchTipsOpen = getDefaultStore().get(AppModel.isSearchTipsOpen);
                if (isSearchTipsOpen) {
                    // Close search tips
                    AppModel.closeSearchTips();

                    // Focus the input directly using the ref
                    setTimeout(() => {
                        inputRef.current?.focus();
                    }, 50);

                    return true;
                }

                // If history dropdown is open, close it
                if (isHistoryOpen) {
                    setIsHistoryOpen(false);
                    return true;
                }

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

                {isHistoryOpen && (
                    <SearchHistoryDropdown
                        ref={dropdownRef}
                        onClose={() => setIsHistoryOpen(false)}
                        onSelect={onValueChange}
                        inputRef={inputRef}
                    />
                )}
            </div>
        </div>
    );
};
