import { arrow, autoUpdate, flip, FloatingArrow, offset, Placement, shift, useFloating } from "@floating-ui/react";
import { Lightbulb } from "lucide-react";
import React, { useRef } from "react";
import { ElementHighlight } from "./element-highlight";

interface ElementTipProps {
    targetElement: HTMLElement | null;
    tipContent: React.ReactNode;
    placement?: Placement;
    padding?: number;
    overlayColor?: string;
    borderRadius?: number;
    allowInteraction?: boolean;
    zIndex?: number;
    onClick?: () => void;
    onClose?: () => void;
}

export const ElementTip: React.FC<ElementTipProps> = ({
    targetElement,
    tipContent,
    placement = "bottom",
    padding = 10,
    overlayColor = "rgba(0, 0, 0, 0.5)",
    borderRadius = 8,
    allowInteraction = true,
    zIndex = 1000,
    onClick,
    onClose,
}) => {
    const arrowRef = useRef<SVGSVGElement>(null);

    // Set up floating UI
    const floatingContext = useFloating({
        elements: {
            reference: targetElement,
        },
        placement,
        whileElementsMounted: autoUpdate,
        middleware: [
            // Add padding to the offset to account for the highlight padding and arrow size
            offset(padding + 20), // Further increased distance from the target element
            flip(), // Flip to the opposite side if there's not enough space
            shift(), // Shift the tooltip if there's not enough space
            arrow({
                element: arrowRef,
                padding: 8, // Increased padding to avoid corners
            }),
        ],
    });

    // Destructure what we need from the floating context
    const { x, y, strategy, refs, context } = floatingContext;

    // If there's no target element, don't render anything
    if (!targetElement) {
        return null;
    }

    return (
        <>
            {/* Highlight component */}
            <ElementHighlight
                targetElement={targetElement}
                padding={padding}
                overlayColor={overlayColor}
                borderRadius={borderRadius}
                allowInteraction={allowInteraction}
                zIndex={zIndex}
                onClick={onClick}
            />

            {/* Tooltip */}
            <div
                ref={floatingContext.refs.setFloating}
                className="bg-panel text-primary rounded-lg shadow-lg max-w-xs border border-border"
                style={{
                    position: strategy,
                    top: y ?? 0,
                    left: x ?? 0,
                    zIndex: zIndex + 1,
                    width: "max-content",
                    maxWidth: "300px",
                }}
            >
                {/* Close button */}
                <button
                    className="absolute top-2 right-2 text-primary hover:text-accent cursor-pointer"
                    onClick={onClose}
                    aria-label="Close tip"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                    >
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>

                {/* Tip header with icon */}
                <div className="p-4 pb-2 flex items-start gap-3">
                    <div className="text-accent">
                        <Lightbulb size={20} />
                    </div>
                    <div className="flex-1">{tipContent}</div>
                </div>

                {/* Action buttons */}
                <div className="border-t border-border p-3 flex justify-end">
                    <button
                        className="bg-accent text-white px-3 py-1 rounded hover:bg-accent/90 text-sm cursor-pointer"
                        onClick={onClose}
                    >
                        Close
                    </button>
                </div>

                {/* Arrow */}
                <FloatingArrow
                    ref={arrowRef}
                    context={floatingContext.context}
                    fill="var(--color-panel)"
                    stroke="rgba(255,255,255,0.2)"
                    strokeWidth={1}
                    height={14}
                    width={28}
                    tipRadius={1}
                />
            </div>
        </>
    );
};
