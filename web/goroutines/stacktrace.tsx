import React from "react";
import { CodeLinkType, GoRoutinesModel } from "./goroutines-model";

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
    if ((simpleMode === "simplified" || simpleMode === "simplified:files") && canUseSimplifiedView) {
        // For now, both simplified modes use the same component
        // In the future, "simplified:files" will have its own implementation
        return <SimplifiedStackTrace goroutine={goroutine} model={model} linkType={linkType} />;
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
}

// Helper function to split package path and highlight only the last part
const HighlightLastPackagePart: React.FC<{ packagePath: string }> = ({ packagePath }) => {
    const parts = packagePath.split("/");
    const lastPart = parts.pop() || "";
    const prefix = parts.length > 0 ? parts.join("/") + "/" : "";

    return (
        <>
            <span className="text-secondary">{prefix}</span>
            <span className="text-primary">{lastPart}</span>
        </>
    );
};

const SimplifiedStackTrace: React.FC<SimplifiedStackTraceProps> = ({ goroutine, model, linkType }) => {
    return (
        <div className="text-xs text-primary bg-panel py-1 px-0 rounded font-mono">
            {goroutine.parsedframes.map((frame, index) => (
                <React.Fragment key={index}>
                    <div
                        className={
                            frame.isimportant
                                ? "border-l-[5px] border-l-border pl-3"
                                : "border-l-[5px] border-l-transparent pl-3"
                        }
                    >
                        <div>
                            <HighlightLastPackagePart packagePath={frame.package} />
                            <span className="text-primary">.{frame.funcname}()</span>
                        </div>
                        {/* Only show file line for important frames */}
                        {frame.isimportant && (
                            <div className="ml-4">
                                {linkType ? (
                                    <a
                                        href={model.generateCodeLink(frame.filepath, frame.linenumber, linkType)}
                                        className="cursor-pointer hover:text-blue-500 hover:underline text-secondary transition-colors duration-150"
                                    >
                                        {frame.filepath}:{frame.linenumber}
                                    </a>
                                ) : (
                                    <span>
                                        {frame.filepath}:{frame.linenumber}
                                    </span>
                                )}
                            </div>
                        )}
                    </div>
                </React.Fragment>
            ))}

            {goroutine.createdbygoid && goroutine.createdbyframe && (
                <React.Fragment>
                    <div
                        className={
                            goroutine.createdbyframe.isimportant
                                ? "border-l-[5px] border-l-border pl-3"
                                : "border-l-[5px] border-l-transparent pl-3"
                        }
                    >
                        <div>
                            <span className="text-secondary">created in goroutine {goroutine.createdbygoid} by </span>
                            <HighlightLastPackagePart packagePath={goroutine.createdbyframe.package} />
                            <span className="text-primary">.{goroutine.createdbyframe.funcname}</span>
                        </div>
                        {/* Only show file line for important frames */}
                        {goroutine.createdbyframe.isimportant && (
                            <div className="ml-4">
                                {linkType ? (
                                    <a
                                        href={model.generateCodeLink(
                                            goroutine.createdbyframe.filepath,
                                            goroutine.createdbyframe.linenumber,
                                            linkType
                                        )}
                                        className="cursor-pointer hover:text-blue-500 hover:underline text-secondary transition-colors duration-150"
                                    >
                                        {goroutine.createdbyframe.filepath}:{goroutine.createdbyframe.linenumber}
                                    </a>
                                ) : (
                                    <span>
                                        {goroutine.createdbyframe.filepath}:{goroutine.createdbyframe.linenumber}
                                    </span>
                                )}
                            </div>
                        )}
                    </div>
                </React.Fragment>
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
