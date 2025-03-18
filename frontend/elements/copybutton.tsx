import { Tooltip } from "@/elements/tooltip";
import { cn } from "@/util/util";
import { cva, type VariantProps } from "class-variance-authority";
import { Check, Copy } from "lucide-react";
import { useCallback, useState } from "react";

const copyButtonVariants = cva(
    "p-1 rounded transition-colors cursor-pointer", 
    {
        variants: {
            variant: {
                muted: "text-muted hover:text-primary hover:bg-buttonhover",
                primary: "text-primary hover:text-primary/80",
            },
            copied: {
                true: "",
                false: "",
            }
        },
        compoundVariants: [
            {
                variant: "muted",
                copied: true,
                className: "text-success hover:text-success/80",
            },
            // No need for primary + copied compound variant since it doesn't change styles
        ],
        defaultVariants: {
            variant: "muted",
            copied: false,
        }
    }
);

interface CopyButtonProps extends VariantProps<typeof copyButtonVariants> {
    className?: string;
    size?: number;
    tooltipText?: string;
    successTooltipText?: string;
    onCopy: () => void | Promise<void>;
}

export function CopyButton({
    className,
    size = 16,
    tooltipText = "Copy to clipboard",
    successTooltipText = "Copied!",
    variant = "muted",
    onCopy,
}: CopyButtonProps) {
    const [copied, setCopied] = useState(false);

    const handleCopy = useCallback(async () => {
        try {
            // Call the onCopy callback
            await onCopy();
            
            setCopied(true);
            
            // Reset after 2 seconds
            setTimeout(() => {
                setCopied(false);
            }, 2000);
        } catch (error) {
            console.error("Failed to copy text:", error);
        }
    }, [onCopy]);

    return (
        <Tooltip content={copied ? successTooltipText : tooltipText}>
            <button
                onClick={handleCopy}
                className={cn(
                    copyButtonVariants({
                        variant,
                        copied,
                        className
                    })
                )}
                aria-label={copied ? "Copied" : "Copy to clipboard"}
            >
                {copied ? <Check size={size} /> : <Copy size={size} />}
            </button>
        </Tooltip>
    );
}
