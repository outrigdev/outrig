// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tag } from "@/elements/tag";
import { TimestampDot } from "@/elements/timestampdot";
import { SearchFilter } from "@/searchfilter/searchfilter";
import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { checkKeyPressed } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { WatchesModel } from "./watches-model";

// Constants for watch flags (matching the Go constants)
const WatchFlag_Push = 1 << 5; // 32
const WatchFlag_Counter = 1 << 6; // 64
const WatchFlag_Atomic = 1 << 7; // 128
const WatchFlag_Sync = 1 << 8; // 256
const WatchFlag_Func = 1 << 9; // 512
const WatchFlag_Hook = 1 << 10; // 1024
const WatchFlag_Settable = 1 << 11; // 2048
const WatchFlag_JSON = 1 << 12; // 4096
const WatchFlag_GoFmt = 1 << 13; // 8192

// Kind mask (lower 5 bits of flags)
const KindMask = 0x1f;

// Go reflect.Kind constants
enum Kind {
    Invalid = 0,
    Bool = 1,
    Int = 2,
    Int8 = 3,
    Int16 = 4,
    Int32 = 5,
    Int64 = 6,
    Uint = 7,
    Uint8 = 8,
    Uint16 = 9,
    Uint32 = 10,
    Uint64 = 11,
    Uintptr = 12,
    Float32 = 13,
    Float64 = 14,
    Complex64 = 15,
    Complex128 = 16,
    Array = 17,
    Chan = 18,
    Func = 19,
    Interface = 20,
    Map = 21,
    Pointer = 22,
    Slice = 23,
    String = 24,
    Struct = 25,
    UnsafePointer = 26,
}

// Get the kind from the flags
function getKind(flags?: number): Kind {
    if (!flags) return Kind.Invalid;
    return flags & (KindMask as Kind);
}

// Get a string representation of the kind
function kindToString(kind: Kind): string {
    return Kind[kind] || "Unknown";
}

// Individual watch view component
interface WatchViewProps {
    watch: WatchSample;
}

const WatchView: React.FC<WatchViewProps> = ({ watch }) => {
    // Format the watch value for display
    const formatValue = (watch: WatchSample) => {
        if (watch.error) {
            return <pre className="text-xs whitespace-pre-wrap font-mono text-error">{watch.error}</pre>;
        }

        if (watch.value == null) {
            return <pre className="text-xs whitespace-pre-wrap font-mono text-muted">null</pre>;
        }

        // Check if the JSON flag is set
        if (watch.flags && watch.flags & WatchFlag_JSON) {
            try {
                const parsed = JSON.parse(watch.value);
                return <pre className="text-xs whitespace-pre-wrap font-mono">{JSON.stringify(parsed, null, 2)}</pre>;
            } catch {
                // If parsing fails, fall back to displaying as is
                return <pre className="text-xs whitespace-pre-wrap font-mono">{watch.value}</pre>;
            }
        }

        // Check if the GoFmt flag is set (formatted by Go's fmt package)
        if (watch.flags && watch.flags & WatchFlag_GoFmt) {
            return <pre className="text-xs whitespace-pre-wrap font-mono">{watch.value}</pre>;
        }

        // Legacy behavior: try to parse JSON if it looks like JSON
        if (watch.value.startsWith("{") || watch.value.startsWith("[")) {
            try {
                const parsed = JSON.parse(watch.value);
                return <pre className="text-xs whitespace-pre-wrap font-mono">{JSON.stringify(parsed, null, 2)}</pre>;
            } catch {
                // If it's not valid JSON, just display as is
            }
        }

        return <pre className="text-xs whitespace-pre-wrap font-mono">{watch.value}</pre>;
    };

    // Get flag tags based on the watch flags
    const getFlagTags = (flags?: number) => {
        if (!flags) return null;

        const tags = [];

        if (flags & WatchFlag_Push) tags.push({ label: "Push", variant: "info" });
        if (flags & WatchFlag_Counter) tags.push({ label: "Counter", variant: "success" });
        if (flags & WatchFlag_Atomic) tags.push({ label: "Atomic", variant: "warning" });
        if (flags & WatchFlag_Sync) tags.push({ label: "Sync", variant: "primary" });
        if (flags & WatchFlag_Func) tags.push({ label: "Func", variant: "secondary" });
        if (flags & WatchFlag_Hook) tags.push({ label: "Hook", variant: "link" });
        if (flags & WatchFlag_Settable) tags.push({ label: "Settable", variant: "info" });
        if (flags & WatchFlag_JSON) tags.push({ label: "JSON", variant: "success" });
        if (flags & WatchFlag_GoFmt) tags.push({ label: "GoFmt", variant: "warning" });

        return tags;
    };

    const flagTags = getFlagTags(watch.flags) ?? [];

    return (
        <div className="pl-4 pr-2">
            <div className="flex justify-between items-center py-2">
                <div className="flex items-center gap-2">
                    <div className="relative flex items-center gap-2">
                        <TimestampDot timestamp={watch.ts} />
                        <div className="font-semibold text-primary flex-grow">{watch.name}</div>
                    </div>
                    <div className="text-sm px-2 py-0.5 rounded-md bg-secondary/10 text-secondary font-mono">
                        {watch.type}
                    </div>
                </div>
                <div className="flex items-center gap-1.5">
                    {/* Display tags with # prefix if they exist */}
                    {watch.tags &&
                        watch.tags.length > 0 &&
                        watch.tags.map((tag, index) => (
                            <Tag key={`tag-${index}`} label={`#${tag}`} isSelected={false} variant="accent" />
                        ))}
                    {flagTags.map((tag, index) => (
                        <span key={index} className="-translate-y-0.5">
                            <Tag label={tag.label} isSelected={false} variant="secondary" />
                        </span>
                    ))}
                    <CopyButton
                        size={14}
                        tooltipText="Copy value"
                        onCopy={() => {
                            if (watch.value) {
                                navigator.clipboard.writeText(watch.value);
                            }
                        }}
                    />
                </div>
            </div>
            <div className="text-sm text-primary pb-2">{formatValue(watch)}</div>
            {(watch.len != null || watch.cap != null || (watch.waittime != null && watch.waittime > 0)) && (
                <div className="pb-2 flex gap-3">
                    {watch.len != null && <span className="text-xs text-muted">Length: {watch.len}</span>}
                    {watch.cap != null && <span className="text-xs text-muted">Capacity: {watch.cap}</span>}
                    {watch.waittime != null && watch.waittime > 0 && (
                        <span className="text-xs text-warning">Wait time: {watch.waittime}μs</span>
                    )}
                </div>
            )}
        </div>
    );
};

// Watches filters component
interface WatchesFiltersProps {
    model: WatchesModel;
}

const WatchesFilters: React.FC<WatchesFiltersProps> = ({ model }) => {
    const [search, setSearch] = useAtom(model.searchTerm);
    const searchResultInfo = useAtomValue(model.searchResultInfo);
    const resultCount = useAtomValue(model.resultCount);
    const errorSpans = searchResultInfo.errorSpans || [];

    return (
        <div className="py-1 px-1 border-b border-border">
            <div className="flex items-center justify-between">
                <SearchFilter
                    value={search}
                    onValueChange={(value) => {
                        setSearch(value);
                        model.updateSearchTerm(value);
                    }}
                    placeholder="Filter watches..."
                    autoFocus={true}
                    errorSpans={errorSpans}
                    onOutrigKeyDown={(keyEvent) => {
                        if (checkKeyPressed(keyEvent, "PageUp")) {
                            model.pageUp();
                            return true;
                        }
                        if (checkKeyPressed(keyEvent, "PageDown")) {
                            model.pageDown();
                            return true;
                        }
                        return false;
                    }}
                />

                {/* Search stats */}
                <div className="text-xs text-muted mr-2 select-none">
                    <span>
                        {resultCount}/{searchResultInfo.totalCount}
                    </span>
                </div>

                <AutoRefreshButton autoRefreshAtom={model.autoRefresh} onToggle={() => model.toggleAutoRefresh()} />
                <RefreshButton
                    isRefreshingAtom={model.isRefreshing}
                    onRefresh={() => model.refresh()}
                    tooltipContent="Refresh watches"
                    size={16}
                />
            </div>
        </div>
    );
};

// Content component that displays the watches
interface WatchesContentProps {
    model: WatchesModel;
}

const WatchesContent: React.FC<WatchesContentProps> = ({ model }) => {
    const filteredWatches = useAtomValue(model.filteredWatches);
    const isRefreshing = useAtomValue(model.isRefreshing);
    const search = useAtomValue(model.searchTerm);
    const contentRef = useRef<HTMLDivElement>(null);
    const [showEmptyMessage, setShowEmptyMessage] = useState(false);

    // Set the content ref in the model when it changes
    useEffect(() => {
        model.setContentRef(contentRef);
    }, [model]);

    // Set a timeout to show empty message after component mounts or when filtered watches change
    useEffect(() => {
        if (filteredWatches.length === 0 && !isRefreshing) {
            const timer = setTimeout(() => {
                setShowEmptyMessage(true);
            }, EmptyMessageDelayMs);
            
            return () => clearTimeout(timer);
        } else {
            setShowEmptyMessage(false);
        }
    }, [filteredWatches.length, isRefreshing]);

    return (
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing watches...</span>
                    </div>
                </div>
            ) : filteredWatches.length === 0 && showEmptyMessage ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search ? "no watches match the filter" : "no watches"}
                </div>
            ) : (
                <div>
                    {filteredWatches.map((watch, index) => (
                        <React.Fragment key={watch.name}>
                            <WatchView watch={watch} />
                            {/* Add divider after each watch except the last one */}
                            {index < filteredWatches.length - 1 && (
                                <div
                                    className="h-px bg-border my-2 w-full"
                                ></div>
                            )}
                        </React.Fragment>
                    ))}
                </div>
            )}
        </div>
    );
};

// Main watches component that composes the sub-components
interface WatchesProps {
    appRunId: string;
}

export const Watches: React.FC<WatchesProps> = ({ appRunId }) => {
    const model = useOutrigModel(WatchesModel, appRunId);

    if (!model) {
        return null;
    }

    return (
        <div className="w-full h-full flex flex-col">
            <WatchesFilters model={model} />
            <WatchesContent model={model} />
        </div>
    );
};
