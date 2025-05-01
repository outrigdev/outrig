// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { cn } from "@/util/util";
import React from "react";

interface ErrorBoundaryProps {
    children: React.ReactNode;
    className?: string;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

/**
 * ErrorBoundary component that catches JavaScript errors in its child component tree,
 * logs them, and displays a fallback UI with the error message.
 */
export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
        console.error("ErrorBoundary caught an error:", error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div
                    className={cn("p-4 rounded", "bg-opacity-10", "text-error font-mono text-sm", this.props.className)}
                >
                    <pre className="whitespace-pre-wrap overflow-auto">
                        <span className="font-bold">
                            Error: {this.state.error?.message || "An unknown error occurred"}
                        </span>
                        {this.state.error?.stack && (
                            <>
                                <br />
                                <br />
                                {this.state.error.stack}
                            </>
                        )}
                    </pre>
                </div>
            );
        }

        return this.props.children;
    }
}
