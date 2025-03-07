import { useState } from "react";
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

interface TooltipProps {
    children: React.ReactNode;
    content: string;
    placement?: "top" | "bottom" | "left" | "right";
}

export function Tooltip({ children, content, placement = "top" }: TooltipProps) {
    const [isOpen, setIsOpen] = useState(false);

    const { refs, floatingStyles, context } = useFloating({
        open: isOpen,
        onOpenChange: setIsOpen,
        placement,
        middleware: [offset(5), flip(), shift()],
        whileElementsMounted: autoUpdate,
    });

    const hover = useHover(context);
    const { getReferenceProps, getFloatingProps } = useInteractions([hover]);

    return (
        <>
            <div ref={refs.setReference} {...getReferenceProps()}>
                {children}
            </div>
            {isOpen && (
                <FloatingPortal>
                    <div
                        ref={refs.setFloating}
                        style={floatingStyles}
                        {...getFloatingProps()}
                        className="bg-panel border border-border rounded-md px-2 py-1 text-xs text-primary shadow-md z-50"
                    >
                        {content}
                    </div>
                </FloatingPortal>
            )}
        </>
    );
}
