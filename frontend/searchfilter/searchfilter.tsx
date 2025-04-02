import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { Filter } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { DELIMITER_PAIRS, SPECIAL_CHARS, handleDelimiter, handleSpecialChar } from "./searchfilter-helpers";

interface SearchFilterProps {
    value: string;
    onValueChange: (value: string) => void;
    placeholder?: string;
    autoFocus?: boolean;
    onOutrigKeyDown?: (keyEvent: OutrigKeyboardEvent) => boolean;
    className?: string;
}

export const SearchFilter: React.FC<SearchFilterProps> = ({
    value,
    onValueChange,
    placeholder = "Filter...",
    autoFocus = false,
    onOutrigKeyDown,
    className = "",
}) => {
    // Handle keydown events for the search filter
    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        const input = e.currentTarget;
        const key = e.key;
        
        // Only process if selection is collapsed (no text is selected)
        if (input.selectionStart === input.selectionEnd) {
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
        </div>
    );
};
