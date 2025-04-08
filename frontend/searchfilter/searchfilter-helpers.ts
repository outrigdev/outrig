// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Helper functions for the SearchFilter component

// Pairs of opening and closing delimiters that should be auto-closed
export const DELIMITER_PAIRS: Record<string, string> = {
    "'": "'",
    '"': '"',
    "/": "/",
    "(": ")",
};

// Special characters that have specific behavior in the search grammar
export const SPECIAL_CHARS = [
    "~", // Fuzzy search prefix
    "$", // Field prefix
    ":", // Field separator
];

// Special case checks for various grammar patterns
const isCaseSensitiveRegexStart = (text: string, cursorPos: number): boolean => {
    // Check if we're right after 'c/'
    if (cursorPos < 2) return false;

    return text.substring(cursorPos - 2, cursorPos) === "c/";
};

const isFuzzySearchStart = (text: string, cursorPos: number): boolean => {
    // Check if we're right after '~'
    if (cursorPos < 1) return false;

    return text[cursorPos - 1] === "~";
};

const isFieldPrefixStart = (text: string, cursorPos: number): boolean => {
    // Check if we're right after '$'
    if (cursorPos < 1) return false;

    return text[cursorPos - 1] === "$";
};

const isInFieldName = (text: string, cursorPos: number): boolean => {
    // Check if we're in a field name (after $ but before :)
    if (cursorPos < 1) return false;

    // Find the last $ before cursor
    const lastDollarPos = text.lastIndexOf("$", cursorPos - 1);
    if (lastDollarPos === -1) return false;

    // Check if there's a colon between $ and cursor
    const colonPos = text.indexOf(":", lastDollarPos);
    return colonPos === -1 || colonPos >= cursorPos;
};

/**
 * Checks if the cursor is already inside a pair of delimiters
 */
export const isCursorInsideDelimiters = (
    text: string,
    cursorPos: number,
    openChar: string,
    closeChar: string
): boolean => {
    // If cursor is at the beginning or end, it can't be inside delimiters
    if (cursorPos <= 0 || cursorPos >= text.length) return false;

    // Check if cursor is directly after an opening delimiter
    const charBeforeCursor = text[cursorPos - 1];

    // If the character before cursor is the opening delimiter
    // and the character at cursor is the closing delimiter,
    // then we're between a pair of delimiters
    if (charBeforeCursor === openChar && text[cursorPos] === closeChar) {
        return true;
    }

    // More complex check for nested delimiters would go here
    // but for simplicity we'll use this basic check for now

    return false;
};

/**
 * Handles special character behavior in the search filter
 * Returns true if the event was handled, false otherwise
 */
export const handleSpecialChar = (
    e: React.KeyboardEvent,
    input: HTMLInputElement,
    onValueChange: (value: string) => void
): boolean => {
    const cursorPos = input.selectionStart;
    if (cursorPos === null) return false;

    const text = input.value;
    const key = e.key;

    // Handle fuzzy search with quotes
    if ((key === "'" || key === '"') && isFuzzySearchStart(text, cursorPos)) {
        // We're typing a quote right after '~', auto-close it
        e.preventDefault();
        const newValue = text.substring(0, cursorPos) + key + key + text.substring(cursorPos);

        onValueChange(newValue);

        // Set cursor position between the quotes
        setTimeout(() => {
            input.setSelectionRange(cursorPos + 1, cursorPos + 1);
        }, 0);

        return true;
    }

    // Handle colon in field names
    if (key === ":" && isInFieldName(text, cursorPos)) {
        // We're typing a colon in a field name, just let it through
        return false;
    }

    return false;
};

/**
 * Checks if the cursor is at a position where a new token would start
 * This helps determine if we should auto-close a delimiter
 */
const isAtNewTokenStart = (text: string, cursorPos: number): boolean => {
    // If cursor is at the beginning, it's a new token
    if (cursorPos <= 0) return true;
    
    // Check if the character before cursor is whitespace or a pipe (|)
    const charBeforeCursor = text[cursorPos - 1];
    return /\s/.test(charBeforeCursor) || charBeforeCursor === '|';
};

/**
 * Checks if the cursor is already inside an unclosed delimiter
 * For example, if we have "/log" and cursor is after "g", we're inside an unclosed "/"
 */
const isInsideUnclosedDelimiter = (text: string, cursorPos: number, openChar: string, closeChar: string): boolean => {
    // If we're at the start of a new token, we're not inside an unclosed delimiter
    if (isAtNewTokenStart(text, cursorPos)) return false;
    
    const textBeforeCursor = text.substring(0, cursorPos);
    
    // Find the start of the current token (last whitespace or pipe before cursor)
    let tokenStartPos = textBeforeCursor.search(/[\s|][^\s|]*$/);
    if (tokenStartPos === -1) {
        tokenStartPos = 0;
    } else {
        tokenStartPos += 1; // Skip the whitespace or pipe
    }
    
    // Only consider text in the current token
    const currentToken = textBeforeCursor.substring(tokenStartPos);
    
    // Count unmatched delimiters in the current token
    let openCount = 0;
    let closeCount = 0;
    let escaped = false;
    
    for (let i = 0; i < currentToken.length; i++) {
        const char = currentToken[i];
        
        if (escaped) {
            // Skip escaped characters
            escaped = false;
            continue;
        }
        
        if (char === "\\") {
            escaped = true;
            continue;
        }
        
        if (char === openChar) {
            openCount++;
        } else if (char === closeChar) {
            closeCount++;
        }
    }
    
    // If we have more opening delimiters than closing ones, we're inside an unclosed delimiter
    return openCount > closeCount;
};

/**
 * Handles wrapping selected text with delimiters
 * Returns true if the event was handled, false otherwise
 */
export const handleSelectionWrapping = (
    e: React.KeyboardEvent,
    input: HTMLInputElement,
    openChar: string,
    closeChar: string,
    onValueChange: (value: string) => void
): boolean => {
    const selectionStart = input.selectionStart;
    const selectionEnd = input.selectionEnd;
    
    // Only process if text is selected
    if (selectionStart === null || selectionEnd === null || selectionStart === selectionEnd) {
        return false;
    }
    
    const text = input.value;
    const selectedText = text.substring(selectionStart, selectionEnd);
    
    // Wrap the selected text with delimiters
    const newValue = 
        text.substring(0, selectionStart) + 
        openChar + selectedText + closeChar + 
        text.substring(selectionEnd);
    
    e.preventDefault();
    onValueChange(newValue);
    
    // Position cursor right before the closing delimiter
    setTimeout(() => {
        input.setSelectionRange(selectionEnd + 1, selectionEnd + 1);
    }, 0);
    
    return true;
};

/**
 * Handles delimiter auto-closing and skipping
 * Returns true if the event was handled, false otherwise
 */
export const handleDelimiter = (
    e: React.KeyboardEvent,
    input: HTMLInputElement,
    openChar: string,
    closeChar: string,
    onValueChange: (value: string) => void
): boolean => {
    const cursorPos = input.selectionStart;
    if (cursorPos === null) return false;

    const text = input.value;

    // Case 1: Check if we should skip over an existing closing delimiter
    if (openChar === e.key && cursorPos < text.length && text[cursorPos] === closeChar) {
        // If the next character is already the closing delimiter, just move the cursor past it
        e.preventDefault();
        input.setSelectionRange(cursorPos + 1, cursorPos + 1);
        return true;
    }

    // Case 2: Auto-close the delimiter
    if (openChar === e.key) {
        // Check if we're already inside an unclosed delimiter of the same type
        // For example, if we have "/log" and type another "/", we don't want to auto-close
        if (isInsideUnclosedDelimiter(text, cursorPos, openChar, closeChar)) {
            return false; // Let the default behavior handle it
        }
        // Special handling for case-sensitive regex (c/)
        if (openChar === "/" && isCaseSensitiveRegexStart(text, cursorPos)) {
            // We're typing a slash right after 'c', so this is a case-sensitive regex start
            // Auto-close it with another slash
            e.preventDefault();
            const newValue = text.substring(0, cursorPos) + "/" + text.substring(cursorPos);

            onValueChange(newValue);

            // Set cursor position between the delimiters
            setTimeout(() => {
                input.setSelectionRange(cursorPos + 1, cursorPos + 1);
            }, 0);

            return true;
        }

        // Don't auto-close if we're inside a word
        const isInsideWord =
            cursorPos > 0 && /\w/.test(text[cursorPos - 1]) && cursorPos < text.length && /\w/.test(text[cursorPos]);

        if (isInsideWord) return false;

        // Check if we're between a pair of delimiters (e.g., cursor is between quotes)
        if (isCursorInsideDelimiters(text, cursorPos, openChar, closeChar)) {
            return false;
        }

        // Auto-close the delimiter
        e.preventDefault();
        const newValue = text.substring(0, cursorPos) + openChar + closeChar + text.substring(cursorPos);

        onValueChange(newValue);

        // Set cursor position between the delimiters
        setTimeout(() => {
            input.setSelectionRange(cursorPos + 1, cursorPos + 1);
        }, 0);

        return true;
    }

    return false;
};
