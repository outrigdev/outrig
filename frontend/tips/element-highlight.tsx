import React, { useEffect, useState } from "react";
import { HighlightOverlay } from "./highlight-overlay";

interface ElementHighlightProps {
    targetElement: HTMLElement | null;
    padding?: number;
    overlayColor?: string;
    borderRadius?: number;
    allowInteraction?: boolean;
    zIndex?: number;
    onClick?: () => void;
    onHighlightClick?: () => void;
}

export const ElementHighlight: React.FC<ElementHighlightProps> = ({
    targetElement,
    padding = 10,
    overlayColor,
    borderRadius,
    allowInteraction = true,
    zIndex,
    onClick,
    onHighlightClick,
}) => {
    const [targetRect, setTargetRect] = useState<DOMRect | null>(null);

    // Update the target rect when the component mounts, when the target element changes,
    // or when the window resizes
    useEffect(() => {
        if (!targetElement) {
            setTargetRect(null);
            return;
        }

        const updateTargetRect = () => {
            const rect = targetElement.getBoundingClientRect();
            
            // Apply padding to the rect
            setTargetRect({
                top: rect.top - padding,
                left: rect.left - padding,
                width: rect.width + (padding * 2),
                height: rect.height + (padding * 2),
                bottom: rect.bottom + padding,
                right: rect.right + padding,
                x: rect.x - padding,
                y: rect.y - padding,
                toJSON: rect.toJSON
            });
        };

        // Initial update
        updateTargetRect();

        // Set up resize observer for the element
        const resizeObserver = new ResizeObserver(updateTargetRect);
        resizeObserver.observe(targetElement);

        // Set up window resize listener
        window.addEventListener('resize', updateTargetRect);
        
        // Set up scroll listener (for when the element moves due to scrolling)
        window.addEventListener('scroll', updateTargetRect, true);
        
        return () => {
            resizeObserver.unobserve(targetElement);
            resizeObserver.disconnect();
            window.removeEventListener('resize', updateTargetRect);
            window.removeEventListener('scroll', updateTargetRect, true);
        };
    }, [targetElement, padding]);

    // If there's no target rect, don't render anything
    if (!targetRect) {
        return null;
    }

    return (
        <HighlightOverlay
            targetRect={targetRect}
            overlayColor={overlayColor}
            borderRadius={borderRadius}
            allowInteraction={allowInteraction}
            zIndex={zIndex}
            onClick={onClick}
            onHighlightClick={onHighlightClick}
        />
    );
};