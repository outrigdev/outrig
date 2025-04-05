import {
    autoUpdate,
    flip,
    FloatingPortal,
    offset,
    shift,
    useClick,
    useDismiss,
    useFloating,
    useInteractions,
    useRole,
} from "@floating-ui/react";
import { ChevronDown } from "lucide-react";
import React, { useState } from "react";
import { cn } from "../util/util";

interface DropdownOption {
    value: string;
    label: string;
}

interface DropdownProps {
    id?: string;
    value: string;
    onChange: (value: string) => void;
    options: DropdownOption[];
    label?: string;
    className?: string;
}

export const Dropdown: React.FC<DropdownProps> = ({ id, value, onChange, options, label, className }) => {
    const [isOpen, setIsOpen] = useState(false);

    // Find the selected option label
    const selectedOption = options.find((option) => option.value === value);
    const selectedLabel = selectedOption ? selectedOption.label : "";

    // Set up floating UI
    const { refs, floatingStyles, context } = useFloating({
        open: isOpen,
        onOpenChange: setIsOpen,
        middleware: [offset(4), flip({ padding: 8 }), shift()],
        whileElementsMounted: autoUpdate,
    });

    // Set up interactions
    const click = useClick(context);
    const dismiss = useDismiss(context);
    const role = useRole(context);

    const { getReferenceProps, getFloatingProps } = useInteractions([click, dismiss, role]);

    return (
        <div className={cn("flex flex-col", className)}>
            {label && (
                <label htmlFor={id} className="mb-1 text-primary">
                    {label}
                </label>
            )}
            <div className="relative">
                <button
                    id={id}
                    ref={refs.setReference}
                    type="button"
                    className="flex items-center justify-between w-full px-3 py-2 text-left bg-buttonbg border border-border rounded-md cursor-pointer text-primary hover:bg-buttonhover"
                    aria-haspopup="listbox"
                    aria-expanded={isOpen}
                    {...getReferenceProps()}
                >
                    <span>{selectedLabel}</span>
                    <ChevronDown className={cn("h-4 w-4 transition-transform", isOpen ? "rotate-180" : "")} />
                </button>

                {isOpen && (
                    <FloatingPortal id="dropdown-portal">
                        <div
                            ref={refs.setFloating}
                            style={{ ...floatingStyles, zIndex: 100 }}
                            className="bg-panel"
                            {...getFloatingProps()}
                        >
                            <ul
                                className="z-[100] w-[var(--reference-width)] bg-panel border border-border rounded-md shadow-lg max-h-60 overflow-auto"
                                role="listbox"
                                style={
                                    {
                                        width: "var(--reference-width)",
                                        "--reference-width": `${(refs.reference.current as HTMLElement)?.offsetWidth || 0}px`,
                                    } as React.CSSProperties
                                }
                            >
                                {options.map((option) => (
                                    <li
                                        key={option.value}
                                        className={cn(
                                            "px-3 py-2 cursor-pointer hover:bg-buttonhover",
                                            option.value === value ? "bg-accentbg/20 text-accent" : "text-primary"
                                        )}
                                        onClick={() => {
                                            onChange(option.value);
                                            setIsOpen(false);
                                        }}
                                        role="option"
                                        aria-selected={option.value === value}
                                    >
                                        {option.label}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </FloatingPortal>
                )}
            </div>
        </div>
    );
};
