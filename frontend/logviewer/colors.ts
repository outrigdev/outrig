// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Color constants for log line coloring
// NOTE: These constants must be kept in sync with server/pkg/searchparser/parser.go
export const ColorNone = 0;
export const ColorRed = 1;
export const ColorGreen = 2;
export const ColorBlue = 3;
export const ColorYellow = 4;
export const ColorPurple = 5;

// Map color values to Tailwind background classes with 20% opacity and hover effects
export function getLogLineColorClass(color: number): string {
    switch (color) {
        case ColorRed:
            return "bg-red-500/20 hover:bg-red-500/30";
        case ColorGreen:
            return "bg-green-500/20 hover:bg-green-500/30";
        case ColorBlue:
            return "bg-blue-500/20 hover:bg-blue-500/30";
        case ColorYellow:
            return "bg-yellow-500/20 hover:bg-yellow-500/30";
        case ColorPurple:
            return "bg-purple-500/20 hover:bg-purple-500/30";
        case ColorNone:
        default:
            return "";
    }
}
