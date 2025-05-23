// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React from "react";

const ANSI_TAILWIND_MAP: { [code: number]: string } = {
    // Reset and modifiers
    0: "reset", // special: clear state
    1: "font-bold",
    2: "opacity-75",
    3: "italic",
    4: "underline",
    8: "invisible",
    9: "line-through",

    // Foreground standard colors
    30: "text-ansi-black",
    31: "text-ansi-red",
    32: "text-ansi-green",
    33: "text-ansi-yellow",
    34: "text-ansi-blue",
    35: "text-ansi-magenta",
    36: "text-ansi-cyan",
    37: "text-ansi-white",

    // Foreground bright colors
    90: "text-ansi-brightblack",
    91: "text-ansi-brightred",
    92: "text-ansi-brightgreen",
    93: "text-ansi-brightyellow",
    94: "text-ansi-brightblue",
    95: "text-ansi-brightmagenta",
    96: "text-ansi-brightcyan",
    97: "text-ansi-brightwhite",

    // Background standard colors
    40: "bg-ansi-black",
    41: "bg-ansi-red",
    42: "bg-ansi-green",
    43: "bg-ansi-yellow",
    44: "bg-ansi-blue",
    45: "bg-ansi-magenta",
    46: "bg-ansi-cyan",
    47: "bg-ansi-white",

    // Background bright colors
    100: "bg-ansi-brightblack",
    101: "bg-ansi-brightred",
    102: "bg-ansi-brightgreen",
    103: "bg-ansi-brightyellow",
    104: "bg-ansi-brightblue",
    105: "bg-ansi-brightmagenta",
    106: "bg-ansi-brightcyan",
    107: "bg-ansi-brightwhite",
};

type InternalStateType = {
    modifiers: Set<string>;
    textColor: string | null;
    bgColor: string | null;
    reverse: boolean;
};

type SegmentType = {
    text: string;
    classes: string;
};

const makeInitialState = (): InternalStateType => ({
    modifiers: new Set<string>(),
    textColor: null,
    bgColor: null,
    reverse: false,
});

const updateStateWithCodes = (state: InternalStateType, codes: number[]): InternalStateType => {
    codes.forEach((code: number) => {
        if (code === 0) {
            state.modifiers.clear();
            state.textColor = null;
            state.bgColor = null;
            state.reverse = false;
            return;
        }
        if (code === 7) {
            state.reverse = true;
            return;
        }
        const tailwindClass = ANSI_TAILWIND_MAP[code];
        if (tailwindClass && tailwindClass !== "reset") {
            if (tailwindClass.startsWith("text-")) {
                state.textColor = tailwindClass;
            } else if (tailwindClass.startsWith("bg-")) {
                state.bgColor = tailwindClass;
            } else {
                state.modifiers.add(tailwindClass);
            }
        }
    });
    return state;
};

const stateToClasses = (state: InternalStateType): string => {
    const classes: string[] = [];
    classes.push(...Array.from(state.modifiers));

    let textColor = state.textColor;
    let bgColor = state.bgColor;
    if (state.reverse) {
        [textColor, bgColor] = [bgColor, textColor];
    }
    if (textColor) classes.push(textColor);
    if (bgColor) classes.push(bgColor);

    return classes.join(" ");
};

// eslint-disable-next-line no-control-regex
const ansiRegex = /\x1b\[([0-9;]+)m/g;

// Regex for tags, kept in sync with server-side TagRegex
const TagRegex = /(^|\s)(#[a-zA-Z0-9][a-zA-Z0-9/_.:-]*)(?=\s|$)/g;

interface AnsiLineProps {
    line: string;
    className?: string;
    onTagClick?: (tag: string, e: React.MouseEvent) => void;
}

const AnsiLine: React.FC<AnsiLineProps> = React.memo(({ line, className = "", onTagClick }) => {
    // Fast path: if no ANSI escapes are found, just render the text.
    if (!line.includes("\x1b[")) {
        if (!onTagClick) {
            return <div className={className}>{line}</div>;
        }

        const nodes: React.ReactNode[] = [];
        TagRegex.lastIndex = 0;
        let last = 0;
        let m: RegExpExecArray | null;
        while ((m = TagRegex.exec(line)) !== null) {
            const [full, leading, tagWithHash] = m;
            const tagStart = m.index + leading.length;
            if (tagStart > last) {
                nodes.push(
                    <span key={nodes.length} className="">
                        {line.slice(last, tagStart)}
                    </span>
                );
            } else if (leading) {
                nodes.push(
                    <span key={nodes.length} className="">
                        {leading}
                    </span>
                );
            }
            nodes.push(
                <span
                    key={nodes.length}
                    className="group cursor-pointer"
                    onClick={(e) => onTagClick(tagWithHash.slice(1), e)}
                >
                    <span className="group-hover:hidden">{tagWithHash}</span>
                    <span className="hidden group-hover:inline text-accent underline">
                        {tagWithHash}
                    </span>
                </span>
            );
            last = m.index + full.length;
        }
        if (last < line.length) {
            nodes.push(
                <span key={nodes.length} className="">
                    {line.slice(last)}
                </span>
            );
        }
        return <div className={className}>{nodes}</div>;
    }

    // Reset regex lastIndex to ensure correct behavior with global regex
    ansiRegex.lastIndex = 0;
    let lastIndex = 0;
    let currentState = makeInitialState();
    const segments: SegmentType[] = [];
    let match: RegExpExecArray | null;
    while ((match = ansiRegex.exec(line)) !== null) {
        if (match.index > lastIndex) {
            segments.push({
                text: line.substring(lastIndex, match.index),
                classes: stateToClasses(currentState),
            });
        }
        const codes = match[1].split(";").map(Number);
        updateStateWithCodes(currentState, codes);
        lastIndex = ansiRegex.lastIndex;
    }
    if (lastIndex < line.length) {
        segments.push({
            text: line.substring(lastIndex),
            classes: stateToClasses(currentState),
        });
    }

    return (
        <div className={className}>
            {segments.map((seg, idx) => {
                if (!onTagClick) {
                    return (
                        <span key={idx} className={seg.classes}>
                            {seg.text}
                        </span>
                    );
                }

                const pieces: React.ReactNode[] = [];
                TagRegex.lastIndex = 0;
                let last = 0;
                let m: RegExpExecArray | null;
                while ((m = TagRegex.exec(seg.text)) !== null) {
                    const [full, leading, tagWithHash] = m;
                    const tagStart = m.index + leading.length;
                    if (tagStart > last) {
                        pieces.push(
                            <span key={`${idx}-${pieces.length}`} className={seg.classes}>
                                {seg.text.slice(last, tagStart)}
                            </span>
                        );
                    } else if (leading) {
                        pieces.push(
                            <span key={`${idx}-${pieces.length}`} className={seg.classes}>
                                {leading}
                            </span>
                        );
                    }
                    pieces.push(
                        <span
                            key={`${idx}-${pieces.length}`}
                            className="group cursor-pointer"
                            onClick={(e) => onTagClick(tagWithHash.slice(1), e)}
                        >
                            <span className={`${seg.classes} group-hover:hidden`}>{tagWithHash}</span>
                            <span className="hidden group-hover:inline text-accent underline">
                                {tagWithHash}
                            </span>
                        </span>
                    );
                    last = m.index + full.length;
                }
                if (last < seg.text.length) {
                    pieces.push(
                        <span key={`${idx}-${pieces.length}`} className={seg.classes}>
                            {seg.text.slice(last)}
                        </span>
                    );
                }
                return pieces;
            })}
        </div>
    );
});

export { AnsiLine };
export type { AnsiLineProps };
