import { cn } from "@/util/util";
import React from "react";
import { CodeLinkType, GoRoutinesModel } from "./goroutines-model";

// Component for displaying a single frame in the simplified stack trace
interface SimplifiedStackFrameProps {
    frame: StackFrame;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
    createdByGoid?: number; // Optional goroutine ID for "created by" frames
    showFileLink?: boolean; // Whether to show the file link separately (true for simplified:files, false for simplified)
}

// Helper function to get just the base filename from a path
const getBaseFileName = (filepath: string): string => {
    const parts = filepath.split("/");
    return parts[parts.length - 1];
};

const SimplifiedStackFrame: React.FC<SimplifiedStackFrameProps> = ({
    frame,
    model,
    linkType,
    createdByGoid,
    showFileLink = true, // Default to showing file link (for backward compatibility)
}) => {
    // Generate the code link if we have a valid linkType
    const codeLink = linkType ? model.generateCodeLink(frame.filepath, frame.linenumber, linkType) : null;

    // Format for the tooltip - (basefilename.go:linenum)
    const fileLocationTip = `(${getBaseFileName(frame.filepath)}:${frame.linenumber})`;

    return (
        <div
            className={
                frame.isimportant ? "border-l-[5px] border-l-border pl-3" : "border-l-[5px] border-l-transparent pl-3"
            }
        >
            <div>
                {/* If not showing file link separately and frame is important, make the entire line clickable */}
                {!showFileLink && codeLink ? (
                    <div className="group relative">
                        {createdByGoid != null && (
                            <div>
                                <span className="text-secondary">created in goroutine {createdByGoid} by </span>
                            </div>
                        )}
                        <a
                            href={codeLink}
                            className={cn("cursor-pointer inline-block", createdByGoid != null ? "pl-4" : "")}
                        >
                            <span className="text-secondary group-hover:text-blue-500 dark:group-hover:text-blue-400  group-hover:decoration-blue-500 dark:group-hover:decoration-blue-400">
                                {frame.package.split("/").slice(0, -1).join("/")}
                                {frame.package.split("/").length > 1 ? "/" : ""}
                            </span>
                            <span className="text-primary group-hover:text-blue-600 dark:group-hover:text-blue-300  group-hover:decoration-blue-600 dark:group-hover:decoration-blue-300">
                                {frame.package.split("/").pop()}.{frame.funcname}
                                {createdByGoid == null ? "()" : ""}
                            </span>
                            <span
                                className="invisible group-hover:visible ml-2 text-secondary absolute italic"
                                style={{ textDecoration: "none" }}
                            >
                                {fileLocationTip}
                            </span>
                        </a>
                    </div>
                ) : (
                    <>
                        {createdByGoid != null && (
                            <div className="text-secondary">created in goroutine {createdByGoid} by </div>
                        )}
                        <HighlightLastPackagePart indent={createdByGoid != null} packagePath={frame.package} />
                        <span className="text-primary">
                            .{frame.funcname}
                            {createdByGoid == null ? "()" : ""}
                        </span>
                    </>
                )}
            </div>
            {/* Only show file line for important frames and when showFileLink is true */}
            {frame.isimportant && showFileLink && (
                <FrameFileLink
                    filepath={frame.filepath}
                    linenumber={frame.linenumber}
                    model={model}
                    linkType={linkType}
                />
            )}
        </div>
    );
};

// Component for displaying a link to a code file and line number
interface FrameLinkProps {
    filepath: string;
    linenumber: number;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
}

const FrameFileLink: React.FC<FrameLinkProps> = ({ filepath, linenumber, model, linkType }) => {
    return (
        <div className="ml-4">
            {linkType ? (
                <a
                    href={model.generateCodeLink(filepath, linenumber, linkType)}
                    className="cursor-pointer hover:text-blue-500 text-secondary transition-colors duration-150"
                >
                    {filepath}:{linenumber}
                </a>
            ) : (
                <span>
                    {filepath}:{linenumber}
                </span>
            )}
        </div>
    );
};

// StackTrace component that decides which stack trace view to show
interface StackTraceProps {
    goroutine: ParsedGoRoutine;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
    simpleMode: string;
}

export const StackTrace: React.FC<StackTraceProps> = ({ goroutine, model, linkType, simpleMode }) => {
    // Check if the goroutine is properly parsed for simplified views
    const canUseSimplifiedView = goroutine.parsed && goroutine.parsedframes && goroutine.parsedframes.length > 0;

    // Handle the different modes
    if (simpleMode === "simplified:files" && canUseSimplifiedView) {
        // Show file links separately (original behavior)
        return <SimplifiedStackTrace goroutine={goroutine} model={model} linkType={linkType} showFileLinks={true} />;
    } else if (simpleMode === "simplified" && canUseSimplifiedView) {
        // Don't show file links separately, instead link the function name
        return <SimplifiedStackTrace goroutine={goroutine} model={model} linkType={linkType} showFileLinks={false} />;
    }

    // Default to raw stack trace
    return <RawStackTrace goroutine={goroutine} model={model} linkType={linkType} />;
};

// Component for displaying raw stack trace
interface RawStackTraceProps {
    goroutine: ParsedGoRoutine;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
}

const RawStackTrace: React.FC<RawStackTraceProps> = ({ goroutine, model, linkType }) => {
    if (!goroutine) return null;

    // Split the stacktrace into lines
    const stacktraceLines = goroutine.rawstacktrace.split("\n");

    return (
        <pre className="text-xs text-primary whitespace-pre-wrap bg-panel p-2 rounded">
            {stacktraceLines.map((line: string, index: number) => (
                <StacktraceLine key={index} line={line} model={model} linkType={linkType} />
            ))}
        </pre>
    );
};

// Component for displaying simplified stack trace
interface SimplifiedStackTraceProps {
    goroutine: ParsedGoRoutine;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
    showFileLinks?: boolean; // Whether to show file links separately
}

// Helper function to split package path and highlight only the last part
const HighlightLastPackagePart: React.FC<{ packagePath: string; indent?: boolean }> = ({ packagePath, indent }) => {
    const parts = packagePath.split("/");
    const lastPart = parts.pop() || "";
    const prefix = parts.length > 0 ? parts.join("/") + "/" : "";

    return (
        <>
            <span className={cn("text-secondary", indent ? "pl-4" : null)}>{prefix}</span>
            <span className="text-primary">{lastPart}</span>
        </>
    );
};

const SimplifiedStackTrace: React.FC<SimplifiedStackTraceProps> = ({
    goroutine,
    model,
    linkType,
    showFileLinks = true, // Default to showing file links (for backward compatibility)
}) => {
    return (
        <div className="text-xs text-primary bg-panel py-1 px-0 rounded font-mono">
            {goroutine.parsedframes.map((frame, index) => (
                <SimplifiedStackFrame
                    key={index}
                    frame={frame}
                    model={model}
                    linkType={linkType}
                    showFileLink={showFileLinks}
                />
            ))}

            {goroutine.createdbygoid && goroutine.createdbyframe && (
                <SimplifiedStackFrame
                    frame={goroutine.createdbyframe}
                    model={model}
                    linkType={linkType}
                    createdByGoid={goroutine.createdbygoid}
                    showFileLink={showFileLinks}
                />
            )}
        </div>
    );
};

// Component for a single stacktrace line with optional VSCode link
interface StacktraceLineProps {
    line: string;
    model: GoRoutinesModel;
    linkType: CodeLinkType;
}

const StacktraceLine: React.FC<StacktraceLineProps> = ({ line, model, linkType }) => {
    // Only process lines that might contain file paths
    if (!line.includes(".go:")) {
        return <div>{line}</div>;
    }

    const parsedLine = model.parseStacktraceLine(line);
    if (!parsedLine || linkType == null) {
        return <div>{line}</div>;
    }

    const { filePath, lineNumber } = parsedLine;
    const link = model.generateCodeLink(filePath, lineNumber, linkType);

    if (!link) {
        return <div>{line}</div>;
    }

    // Find the file:line part in the text to make it clickable
    const fileLinePattern = new RegExp(`(${filePath.replace(/\//g, "\\/")}:${lineNumber})`);
    const parts = line.split(fileLinePattern);

    if (parts.length === 1) {
        // Pattern not found, return the line as is
        return <div>{line}</div>;
    }

    return (
        <div>
            {parts.map((part, index) => {
                // If this part matches the file:line pattern, make it a link
                if (part === `${filePath}:${lineNumber}`) {
                    return (
                        <a key={index} href={link} className="group cursor-pointer">
                            <span className="group-hover:text-blue-500 group-hover:underline transition-colors duration-150">
                                {part}
                            </span>
                        </a>
                    );
                }
                return <span key={index}>{part}</span>;
            })}
        </div>
    );
};
