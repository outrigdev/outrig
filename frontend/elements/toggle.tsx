import React from "react";
import { cn } from "../util/util";

interface ToggleProps {
    id?: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
    label?: string;
    className?: string;
}

export const Toggle: React.FC<ToggleProps> = ({ id, checked, onChange, label, className }) => {
    return (
        <div className={cn("flex items-center", className)}>
            <button
                id={id}
                type="button"
                role="switch"
                aria-checked={checked}
                className={cn(
                    "relative inline-flex h-5 w-10 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none",
                    checked ? "bg-accent" : "bg-border"
                )}
                onClick={() => onChange(!checked)}
            >
                <span className="sr-only">Toggle {label}</span>
                <span
                    aria-hidden="true"
                    className={cn(
                        "pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white dark:bg-black shadow-md ring-0 transition duration-200 ease-in-out",
                        checked ? "translate-x-5" : "translate-x-0"
                    )}
                />
            </button>
            {label && (
                <label
                    htmlFor={id}
                    className="ml-3 cursor-pointer select-none text-primary"
                    onClick={() => onChange(!checked)}
                >
                    {label}
                </label>
            )}
        </div>
    );
};
