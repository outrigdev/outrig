// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import {
    FloatingPortal,
    autoUpdate,
    flip,
    offset,
    shift,
    useFloating,
    useHover,
    useInteractions,
} from "@floating-ui/react";
import { useEffect, useRef, useState } from "react";

interface TooltipProps {
    children: React.ReactNode;
    content: React.ReactNode;
    placement?: "top" | "bottom" | "left" | "right";
    forceOpen?: boolean;
    divClassName?: string;
    divStyle?: React.CSSProperties;
    divOnClick?: (e: React.MouseEvent<HTMLDivElement>) => void;
}

export function Tooltip({
    children,
    content,
    placement = "top",
    forceOpen = false,
    divClassName,
    divStyle,
    divOnClick,
}: TooltipProps) {
    const [isOpen, setIsOpen] = useState(forceOpen);
    const [isVisible, setIsVisible] = useState(false);
    const timeoutRef = useRef<number | null>(null);
    const prevForceOpenRef = useRef<boolean>(forceOpen);

    const { refs, floatingStyles, context } = useFloating({
        open: isOpen,
        onOpenChange: (open) => {
            // Don't close if forceOpen is true
            if (!open && forceOpen) {
                return;
            }
            if (open) {
                // When opening, set isOpen immediately but delay visibility
                setIsOpen(true);
                // Clear any existing timeout
                if (timeoutRef.current !== null) {
                    window.clearTimeout(timeoutRef.current);
                }
                // Set a timeout to make it visible after delay
                timeoutRef.current = window.setTimeout(() => {
                    setIsVisible(true);
                }, 300); // 500ms delay before showing
            } else {
                // When closing, keep isOpen true but set visibility to false
                setIsVisible(false);
                // Clear any existing timeout
                if (timeoutRef.current !== null) {
                    window.clearTimeout(timeoutRef.current);
                }
                // Set a timeout to actually close after transition
                timeoutRef.current = window.setTimeout(() => {
                    setIsOpen(false);
                }, 300); // 500ms for fade out transition
            }
        },
        placement,
        middleware: [
            offset(10), // Increased from 5px to 10px for more space
            flip(),
            shift({ padding: 12 }), // Added 12px padding to keep tooltip away from viewport edges
        ],
        whileElementsMounted: autoUpdate,
    });

    // Update isOpen when forceOpen changes
    useEffect(() => {
        if (forceOpen) {
            // When forceOpen becomes true, open the tooltip immediately
            setIsOpen(true);
            setIsVisible(true);

            // Clear any existing timeout
            if (timeoutRef.current !== null) {
                window.clearTimeout(timeoutRef.current);
                timeoutRef.current = null;
            }
        } else {
            // When forceOpen becomes false, close the tooltip
            // Only keep it open if it's being hovered AND forceOpen was previously false
            // (i.e., it was opened by hover, not by forceOpen)
            if (context.open && !prevForceOpenRef.current) {
                // Keep it open if it's being hovered and wasn't forced open before
            } else {
                setIsVisible(false);

                // Clear any existing timeout
                if (timeoutRef.current !== null) {
                    window.clearTimeout(timeoutRef.current);
                }

                // Set a timeout to actually close after transition
                timeoutRef.current = window.setTimeout(() => {
                    setIsOpen(false);
                }, 300);
            }
        }

        // Track previous forceOpen value
        prevForceOpenRef.current = forceOpen;
    }, [forceOpen, context.open]);

    // Clean up timeouts on unmount
    useEffect(() => {
        return () => {
            if (timeoutRef.current !== null) {
                window.clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    const hover = useHover(context);
    const { getReferenceProps, getFloatingProps } = useInteractions([hover]);

    return (
        <>
            <div
                ref={refs.setReference}
                {...getReferenceProps()}
                className={divClassName}
                style={divStyle}
                onClick={divOnClick}
            >
                {children}
            </div>
            {isOpen && (
                <FloatingPortal>
                    <div
                        ref={refs.setFloating}
                        style={{
                            ...floatingStyles,
                            opacity: isVisible ? 1 : 0,
                            transition: "opacity 200ms ease",
                        }}
                        {...getFloatingProps()}
                        className="bg-white dark:bg-[#4a5060] border border-border rounded-md px-2 py-1 text-xs text-primary shadow-xl z-50"
                    >
                        {content}
                    </div>
                </FloatingPortal>
            )}
        </>
    );
}
