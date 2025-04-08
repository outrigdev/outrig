// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import { X } from "lucide-react";
import { useEffect, useState } from "react";

export interface Toast {
    id: string;
    title: string;
    message: string;
    timeout?: number; // milliseconds, null for persistent
}

interface ToastProps {
    toast: Toast;
    onClose: (id: string) => void;
}

export function Toast({ toast, onClose }: ToastProps) {
    const { id, title, message, timeout } = toast;
    const [isVisible, setIsVisible] = useState(true);
    const [isExiting, setIsExiting] = useState(false);
    const [isHovered, setIsHovered] = useState(false);
    const [remainingTime, setRemainingTime] = useState(timeout);
    const [startTime, setStartTime] = useState<number | null>(null);

    // Handle automatic timeout with hover pause
    useEffect(() => {
        if (timeout != null && remainingTime != null && !isHovered) {
            setStartTime(Date.now());
            const timer = setTimeout(() => {
                handleClose();
            }, remainingTime);
            
            return () => {
                clearTimeout(timer);
                if (startTime !== null) {
                    const elapsed = Date.now() - startTime;
                    setRemainingTime(prev => prev != null ? Math.max(0, prev - elapsed) : null);
                }
            };
        }
    }, [timeout, isHovered, remainingTime]);

    // Handle close with animation
    const handleClose = () => {
        setIsExiting(true);
        // Wait for animation to complete before removing
        setTimeout(() => {
            setIsVisible(false);
            onClose(id);
        }, 300); // Match this with CSS transition duration
    };

    if (!isVisible) return null;

    return (
        <div
            className={cn(
                "bg-panel border border-border rounded-md shadow-md p-4 mb-3 w-80 max-w-full",
                "transition-all duration-300 ease-in-out",
                isExiting ? "opacity-0 translate-x-5" : "opacity-100 translate-x-0"
            )}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >
            <div className="flex justify-between items-start">
                <div className="font-medium text-primary">{title}</div>
                <button onClick={handleClose} className="text-secondary hover:text-primary transition-colors">
                    <X size={16} />
                </button>
            </div>
            <div className="text-sm text-secondary mt-1">{message}</div>
        </div>
    );
}

export interface ToastContainerProps {
    toasts: Toast[];
    onClose: (id: string) => void;
}

export function ToastContainer({ toasts, onClose }: ToastContainerProps) {
    if (toasts.length === 0) return null;

    return (
        <div className="fixed bottom-4 right-4 z-50 flex flex-col items-end">
            {toasts.map((toast) => (
                <Toast key={toast.id} toast={toast} onClose={onClose} />
            ))}
        </div>
    );
}
