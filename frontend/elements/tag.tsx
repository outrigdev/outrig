// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import React from "react";

type TagVariant = "primary" | "secondary" | "link" | "info" | "success" | "warning" | "danger" | "accent";

interface TagProps {
    label: string;
    isSelected: boolean;
    onToggle?: () => void;
    variant?: TagVariant;
    count?: number | string;
    className?: string;
    compact?: boolean;
}

export const Tag: React.FC<TagProps> = ({ label, isSelected, onToggle, variant = "primary", count, className, compact = false }) => {
    const baseClasses = cn("text-xs rounded-md transition-colors", getTagStyles(variant, isSelected), className);
    
    const content = (
        <div className="flex items-center h-full">
            <span className={compact ? "px-1.5 py-0.5" : "px-2 py-1"}>{label}</span>
            {count != null && (
                <span className={cn("count font-medium border-l border-current/30", compact ? "px-1 py-0.5" : "px-1.5 py-1")}>{count}</span>
            )}
        </div>
    );
    
    return onToggle ? (
        <button onClick={onToggle} className={cn("cursor-pointer", baseClasses)}>
            {content}
        </button>
    ) : (
        <div className={baseClasses}>
            {content}
        </div>
    );
};

// Helper function to get the appropriate styles based on variant and selection state
function getTagStyles(variant: TagVariant, isSelected: boolean): string {
    if (isSelected) {
        switch (variant) {
            case "primary":
                return "bg-primary/20 text-primary border border-primary/30";
            case "secondary":
                return "bg-secondary/20 text-secondary border border-secondary/30";
            case "link":
                return "bg-blue-500/20 text-blue-500 border border-blue-500/30";
            case "info":
                return "bg-sky-500/20 text-sky-500 border border-sky-500/30";
            case "success":
                return "bg-green-500/20 text-green-500 border border-green-500/30";
            case "warning":
                return "bg-amber-500/20 text-amber-500 border border-amber-500/30";
            case "danger":
                return "bg-red-500/20 text-red-500 border border-red-500/30";
            case "accent":
                return "bg-accent/20 text-accent border border-accent/30";
            default:
                return "bg-primary/20 text-primary border border-primary/30";
        }
    } else {
        switch (variant) {
            case "primary":
                return "bg-primary/10 text-primary/80 border border-primary/20 hover:bg-primary/20";
            case "secondary":
                return "bg-secondary/10 text-secondary border border-secondary/20 hover:bg-secondary/20";
            case "link":
                return "bg-blue-500/10 text-blue-500/80 border border-blue-500/20 hover:bg-blue-500/20";
            case "info":
                return "bg-sky-500/10 text-sky-500/80 border border-sky-500/20 hover:bg-sky-500/20";
            case "success":
                return "bg-green-500/10 text-green-500/80 border border-green-500/20 hover:bg-green-500/20";
            case "warning":
                return "bg-amber-500/10 text-amber-500/80 border border-amber-500/20 hover:bg-amber-500/20";
            case "danger":
                return "bg-red-500/10 text-red-500/80 border border-red-500/20 hover:bg-red-500/20";
            case "accent":
                return "bg-accent/10 text-accent/80 border border-accent/20 hover:bg-accent/20";
            default:
                return "bg-secondary/10 text-secondary border border-secondary/20 hover:bg-secondary/20";
        }
    }
}
