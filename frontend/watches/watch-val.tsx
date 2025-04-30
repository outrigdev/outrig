import { CopyButton } from "@/elements/copybutton";
import { cn } from "@/util/util";
import React from "react";

interface WatchValProps {
    content: string;
    className?: string;
    tooltipText?: string;
    successTooltipText?: string;
    copyButtonSize?: number;
    tag?: string;
}

export const WatchVal: React.FC<WatchValProps> = ({
    content,
    className,
    tooltipText = "Copy to clipboard",
    successTooltipText = "Copied!",
    copyButtonSize = 14,
    tag,
}) => {
    // Split content into lines to handle the tag insertion
    const contentLines = content.split("\n");

    return (
        <div className={cn("relative group rounded-md p-1.5", className)}>
            <pre className={cn("text-xs whitespace-pre-wrap font-mono")}>
                {tag && <span className="inline-block text-xs text-muted select-none">{tag + " "}</span>}
                {contentLines.map((line, index) => (
                    <React.Fragment key={index}>
                        {index > 0 && "\n"}
                        {line}
                    </React.Fragment>
                ))}
            </pre>
            <div className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <CopyButton
                    size={copyButtonSize}
                    tooltipText={tooltipText}
                    successTooltipText={successTooltipText}
                    onCopy={() => navigator.clipboard.writeText(content)}
                    className={cn("bg-background/80")}
                />
            </div>
        </div>
    );
};
