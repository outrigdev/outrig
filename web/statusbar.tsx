import { useSetAtom } from "jotai";
import { Box, CircleDot, List, Wifi } from "lucide-react";
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
import { AppModel } from "./appmodel";

interface TooltipProps {
    children: React.ReactNode;
    content: string;
    placement?: "top" | "bottom" | "left" | "right";
}

function Tooltip({ children, content, placement = "top" }: TooltipProps) {
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

export function StatusBar() {
    const numGoRoutines = 24;
    const numLogLines = 1083;
    const setSelectedTab = useSetAtom(AppModel.selectedTab);

    return (
        <div className="h-6 bg-panel border-t border-border flex items-center justify-between px-2 text-xs text-secondary">
            <div className="flex items-center space-x-4">
                <div className="flex items-center space-x-1">
                    <Box size={12} />
                    <span>appname</span>
                </div>
                <div className="flex items-center space-x-1">
                    <Wifi size={12} />
                    <span>Connected</span>
                </div>
            </div>
            <div className="flex items-center space-x-4">
                <Tooltip content={`${numLogLines} Log Lines`} placement="bottom">
                    <div
                        className="flex items-center space-x-1 cursor-pointer"
                        onClick={() => setSelectedTab("logs")}
                    >
                        <List size={12} />
                        <span>1083</span>
                    </div>
                </Tooltip>
                <Tooltip content={`${numGoRoutines} GoRoutines`} placement="bottom">
                    <div
                        className="flex items-center space-x-1 cursor-pointer"
                        onClick={() => setSelectedTab("goroutines")}
                    >
                        <CircleDot size={12} />
                        <span>24</span>
                    </div>
                </Tooltip>
            </div>
        </div>
    );
}
