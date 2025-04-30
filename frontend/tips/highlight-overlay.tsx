import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import React from "react";
import { createPortal } from "react-dom";

interface HighlightOverlayProps {
    targetRect: DOMRect | { top: number; left: number; width: number; height: number };
    overlayColor?: string;
    borderRadius?: number;
    allowInteraction?: boolean;
    zIndex?: number;
    onClick?: () => void;
    onHighlightClick?: () => void;
}

export const HighlightOverlay: React.FC<HighlightOverlayProps> = ({
    targetRect,
    overlayColor = "rgba(0, 0, 0, 0.6)",
    borderRadius = 8,
    allowInteraction = true,
    zIndex = 1000,
    onClick,
    onHighlightClick,
}) => {
    // Calculate positions for the four overlay divs

    // Check if dark mode is enabled
    const isDarkMode = useAtomValue(AppModel.darkMode);

    // Only keep the dynamic positioning styles inline
    const highlightStyle = {
        top: `${targetRect.top}px`,
        left: `${targetRect.left}px`,
        width: `${targetRect.width}px`,
        height: `${targetRect.height}px`,
        boxShadow: `0 0 0 9999px ${overlayColor}`,
        // Add a slight white tint in dark mode
        backgroundColor: isDarkMode ? "rgba(255, 255, 255, 0.1)" : undefined,
    };

    // Check and create each overlay
    const hasTopOverlay = targetRect.top > 0;
    const topOverlayStyle = {
        top: 0,
        left: 0,
        width: "100%",
        height: `${targetRect.top}px`,
    };

    const hasRightOverlay = targetRect.left + targetRect.width < window.innerWidth;
    const rightOverlayStyle = {
        top: `${targetRect.top}px`,
        left: `${targetRect.left + targetRect.width}px`,
        width: `calc(100% - ${targetRect.left + targetRect.width}px)`,
        height: `${targetRect.height}px`,
    };

    const hasBottomOverlay = targetRect.top + targetRect.height < window.innerHeight;
    const bottomOverlayStyle = {
        top: `${targetRect.top + targetRect.height}px`,
        left: 0,
        width: "100%",
        height: `calc(100% - ${targetRect.top + targetRect.height}px)`,
    };

    const hasLeftOverlay = targetRect.left > 0;
    const leftOverlayStyle = {
        top: `${targetRect.top}px`,
        left: 0,
        width: `${targetRect.left}px`,
        height: `${targetRect.height}px`,
    };

    // Create the overlay content
    const overlayContent = (
        <div
            className="fixed inset-0 pointer-events-none"
            style={{ zIndex }}
        >
            {/* Top overlay - only render if it has positive height */}
            {hasTopOverlay && (
                <div
                    className="fixed bg-transparent pointer-events-auto"
                    style={{ ...topOverlayStyle, zIndex }}
                    onClick={onClick}
                />
            )}

            {/* Right overlay - only render if it has positive width */}
            {hasRightOverlay && (
                <div
                    className="fixed bg-transparent pointer-events-auto"
                    style={{ ...rightOverlayStyle, zIndex }}
                    onClick={onClick}
                />
            )}

            {/* Bottom overlay - only render if it has positive height */}
            {hasBottomOverlay && (
                <div
                    className="fixed bg-transparent pointer-events-auto"
                    style={{ ...bottomOverlayStyle, zIndex }}
                    onClick={onClick}
                />
            )}

            {/* Left overlay - only render if it has positive width */}
            {hasLeftOverlay && (
                <div
                    className="fixed bg-transparent pointer-events-auto"
                    style={{ ...leftOverlayStyle, zIndex }}
                    onClick={onClick}
                />
            )}

            {/* Highlight div - onHighlightClick only works when allowInteraction is false */}
            <div
                className={`fixed ${allowInteraction ? "pointer-events-none" : "pointer-events-auto"}`}
                style={{
                    ...highlightStyle,
                    borderRadius: `${borderRadius}px`,
                    zIndex,
                }}
                onClick={!allowInteraction && onHighlightClick ? onHighlightClick : undefined}
            />
        </div>
    );

    // Get the portal container
    const portalContainer = document.getElementById("highlight-overlay-root");

    // If the portal container doesn't exist yet, return null
    if (!portalContainer) {
        console.log("Highlight overlay portal container not found");
        return null;
    }

    // Otherwise, render through the portal
    return createPortal(overlayContent, portalContainer);
};
