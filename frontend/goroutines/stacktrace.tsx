// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0
import { cn, escapeRegExp } from "@/util/util";
import React, { useState } from "react";
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
                            <span className="text-primary group-hover:text-blue-600 dark:group-hover:text-blue-300 group-hover:font-bold">
                                {frame.package.split("/").pop()}.{frame.funcname}()
                            </span>
                            {/* Only show "in package" if the package name is different from the last part of the path */}
                            {frame.package.split("/").pop() !== frame.package && (
                                <span className="text-secondary ml-1 group-hover:text-blue-600 dark:group-hover:text-blue-300">
                                    in {frame.package}
                                </span>
                            )}
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
                        <span className={cn("text-primary", createdByGoid != null ? "pl-4" : "")}>
                            {frame.package.split("/").pop()}.{frame.funcname}()
                        </span>
                        {/* Only show "in package" if the package name is different from the last part of the path */}
                        {frame.package.split("/").pop() !== frame.package && (
                            <span className="text-secondary ml-1">in {frame.package}</span>
                        )}
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

    // Create the header line in the format Go would use: "goroutine X [state, X minutes]:"
    const headerLine = `goroutine ${goroutine.goid} [${goroutine.rawstate}]:`;

    // Split the stacktrace into lines
    const stacktraceLines = goroutine.rawstacktrace.split("\n");

    return (
        <pre className="text-xs text-primary whitespace-pre-wrap bg-panel p-2 rounded">
            {/* First render the header line */}
            <div>{headerLine}</div>

            {/* Then render the rest of the stack trace */}
            {stacktraceLines.map((line: string, index: number) => (
                <StacktraceLine key={index} line={line} model={model} linkType={linkType} />
            ))}
        </pre>
    );
};

// Helper function to simplify package names based on their source
const simplifyPackageName = (packagePath: string, isSys?: boolean): string => {
    // For golang.org packages, strip the golang.org prefix (even if marked as system)
    if (packagePath.startsWith("golang.org/")) {
        return packagePath.substring("golang.org/".length);
    }

    // For system packages (except golang.org), use the full name
    if (isSys) {
        return packagePath;
    }

    // For GitHub/Bitbucket packages, use just the second part (the repo name)
    if (packagePath.startsWith("github.com/") || packagePath.startsWith("bitbucket.org/")) {
        const parts = packagePath.split("/");
        if (parts.length >= 3) {
            // Return just the repository name (parts[2])
            return parts[2];
        }
    }

    // For gopkg.in packages, take the first part after gopkg.in
    if (packagePath.startsWith("gopkg.in/")) {
        return packagePath.substring("gopkg.in/".length);
    }

    // For internal packages (no domain), use the last part
    if (!packagePath.includes(".")) {
        const parts = packagePath.split("/");
        return parts[parts.length - 1];
    }

    // For anything else, use the full package path
    return packagePath;
};

// Helper function to get the function display name (package.funcname)
const getFunctionDisplayName = (frame: StackFrame): string => {
    const packageName = simplifyPackageName(frame.package, frame.issys);
    return `${packageName}.${frame.funcname}`;
};

// Component for displaying a collapsed section of stack frames
interface CollapsedStackFramesProps {
    frames: StackFrame[];
    onExpand: () => void;
}

const CollapsedStackFrames: React.FC<CollapsedStackFramesProps> = ({ frames, onExpand }) => {
    if (frames.length === 0) return null;

    // Don't collapse if there's only 1 frame
    if (frames.length === 1) {
        return null; // This will be handled by the parent component
    }

    // Use a nicer arrow from Nerd Font
    const arrow = " → ";

    // Get unique package names from the frames (in original order - top to bottom)
    const packageNames = new Set<string>();
    frames.forEach((frame) => {
        const simplifiedName = simplifyPackageName(frame.package, frame.issys);
        packageNames.add(simplifiedName);
    });

    // Convert to array and limit to at most 3 packages
    const uniquePackages = Array.from(packageNames);
    const displayPackages =
        uniquePackages.length <= 3
            ? uniquePackages
            : [uniquePackages[0], uniquePackages[1], "...", uniquePackages[uniquePackages.length - 1]];

    // Format the display text
    const displayText = `... // ${frames.length} frames - ${displayPackages.join(arrow)}`;

    return (
        <div
            className="border-l-[5px] border-l-transparent pl-3 cursor-pointer group"
            onClick={onExpand}
            title="Click to Expand"
        >
            <span className="text-secondary group-hover:text-primary">{displayText}</span>
        </div>
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
    // State to track which sections are expanded
    const [expandedSections, setExpandedSections] = useState<Set<number>>(new Set());

    // Toggle expansion of a section
    const toggleSection = (sectionIndex: number) => {
        const newExpandedSections = new Set(expandedSections);
        if (newExpandedSections.has(sectionIndex)) {
            newExpandedSections.delete(sectionIndex);
        } else {
            newExpandedSections.add(sectionIndex);
        }
        setExpandedSections(newExpandedSections);
    };

    // Group frames into sections (important frames and non-important sections)
    const renderFrames = () => {
        if (!goroutine.parsedframes || goroutine.parsedframes.length === 0) {
            return null;
        }

        const result: React.ReactNode[] = [];
        let currentNonImportantFrames: StackFrame[] = [];
        let sectionIndex = 0;

        // Helper to add non-important frames as a collapsed section
        const addNonImportantSection = () => {
            if (currentNonImportantFrames.length > 0) {
                const currentSectionIndex = sectionIndex++;

                // If expanded or only 1 frame, show all frames individually
                if (expandedSections.has(currentSectionIndex) || currentNonImportantFrames.length === 1) {
                    currentNonImportantFrames.forEach((frame, frameIndex) => {
                        result.push(
                            <SimplifiedStackFrame
                                key={`section-${currentSectionIndex}-frame-${frameIndex}`}
                                frame={frame}
                                model={model}
                                linkType={linkType}
                                showFileLink={showFileLinks}
                            />
                        );
                    });
                } else {
                    // If collapsed and more than 1 frame, show as a single clickable element
                    result.push(
                        <CollapsedStackFrames
                            key={`collapsed-section-${currentSectionIndex}`}
                            frames={currentNonImportantFrames}
                            onExpand={() => toggleSection(currentSectionIndex)}
                        />
                    );
                }
                currentNonImportantFrames = [];
            }
        };

        // Process all frames
        goroutine.parsedframes.forEach((frame, index) => {
            // First frame is always important
            const isFirstFrame = index === 0;
            // Last frame before an important frame is also important (e.g., os.(*File).Read())
            const isLastBeforeImportant =
                index < goroutine.parsedframes.length - 1 && goroutine.parsedframes[index + 1].isimportant;

            if (frame.isimportant || isFirstFrame || isLastBeforeImportant) {
                // Add any accumulated non-important frames first
                addNonImportantSection();

                // Then add this important frame
                result.push(
                    <SimplifiedStackFrame
                        key={`frame-${index}`}
                        frame={frame}
                        model={model}
                        linkType={linkType}
                        showFileLink={showFileLinks}
                    />
                );
            } else {
                // Accumulate non-important frames
                currentNonImportantFrames.push(frame);
            }
        });

        // Add any remaining non-important frames
        addNonImportantSection();

        return result;
    };

    return (
        <div className="text-xs text-primary bg-panel py-1 px-0 rounded font-mono">
            {renderFrames()}

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
    const escapedFilePath = escapeRegExp(filePath);
    const fileLinePattern = new RegExp(`(${escapedFilePath}:${lineNumber})`);
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
                        <a
                            key={index}
                            href={link}
                            className="cursor-pointer hover:text-blue-500 dark:hover:text-blue-300 transition-colors duration-150"
                        >
                            {part}
                        </a>
                    );
                }
                return <span key={index}>{part}</span>;
            })}
        </div>
    );
};
