// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { Modal } from "@/elements/modal";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tag } from "@/elements/tag";
import { TimestampDot } from "@/elements/timestampdot";
import { Tooltip } from "@/elements/tooltip";
import { SearchFilter } from "@/searchfilter/searchfilter";
import { sendClickEvent } from "@/tevent";
import { EmptyMessageDelayMs } from "@/util/constants";
import { useOutrigModel } from "@/util/hooks";
import { checkKeyPressed } from "@/util/keyutil";
import { prettyPrintGoFmt, prettyPrintJson } from "@/util/util";
import { useAtom, useAtomValue } from "jotai";
import React, { useEffect, useRef, useState } from "react";
import { NoWatchesMessage } from "./nowatchmessage";
import { WatchVal } from "./watch-val";
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
    watch: CombinedWatchSample;
    model: WatchesModel;
}

// Component to display watch value and related information
interface WatchValueDisplayProps {
    sample: WatchSample;
}

const WatchValueDisplay: React.FC<WatchValueDisplayProps> = ({ sample }) => {
    // Format the watch value for display
    const formatValue = () => {
        if (sample.error) {
            return <WatchVal content={sample.error} className="text-error" tooltipText="Copy error message" />;
        }

        if (sample.val) {
            // Format based on the sample fmt field
            if (sample.fmt === "json") {
                return <WatchVal content={prettyPrintJson(sample.val)} tooltipText="Copy value" tag={sample.fmt} />;
            } else if (sample.fmt === "gofmt") {
                return <WatchVal content={prettyPrintGoFmt(sample.val)} tooltipText="Copy value" tag={sample.fmt} />;
            } else {
                return <WatchVal content={sample.val} tooltipText="Copy value" tag={sample.fmt} />;
            }
        }

        return <WatchVal content="(no value)" className="text-error" />;
    };

    return (
        <>
            <div className="text-sm text-primary pb-2">
                {formatValue()}
            </div>
            {(sample.len != null || sample.cap != null || (sample.polldur != null && sample.polldur > 2000)) && (
                <div className="pb-2 flex gap-3">
                    {sample.len != null && <span className="text-xs text-muted">Length: {sample.len}</span>}
                    {sample.cap != null && <span className="text-xs text-muted">Capacity: {sample.cap}</span>}
                    {sample.polldur != null && sample.polldur > 2000 && (
                        <span className="text-xs text-warning">
                            Long poll duration: {(sample.polldur / 1000).toFixed(2)}ms
                        </span>
                    )}
                </div>
            )}
        </>
    );
};

const WatchView: React.FC<WatchViewProps> = ({ watch, model }) => {
    // Get tags based on the watch declaration
    const getWatchTags = (decl: WatchDecl) => {
        const tags = [];

        if (decl.counter) tags.push({ label: "Counter", variant: "success" });
        if (decl.invalid) tags.push({ label: "Invalid", variant: "error" });
        if (decl.unregistered) tags.push({ label: "Unregistered", variant: "warning" });

        // Add watchtype as a tag
        if (decl.watchtype) tags.push({ label: decl.watchtype, variant: "info" });

        return tags;
    };

    const watchTags = getWatchTags(watch.decl);

    return (
        <div className="pl-4 pr-2">
            <div className="flex justify-between items-center py-2">
                <div className="flex items-center gap-2">
                    <div className="relative flex items-center gap-2">
                        <TimestampDot timestamp={watch.sample.ts} />
                        <div className="font-semibold text-primary flex-grow">{watch.decl.name}</div>
                    </div>
                    <div className="text-sm px-2 py-0.5 rounded-md bg-secondary/10 text-secondary font-mono">
                        {watch.sample.type}
                    </div>
                </div>
                <div className="flex items-center gap-1.5">
                    {/* Display tags with # prefix if they exist */}
                    {watch.decl.tags &&
                        watch.decl.tags.length > 0 &&
                        watch.decl.tags.map((tag, index) => (
                            <Tag key={`tag-${index}`} label={`#${tag}`} isSelected={false} variant="accent" />
                        ))}
                    {watchTags.map((tag, index) => (
                        <Tag
                            key={`flag-${index}`}
                            label={tag.label}
                            isSelected={false}
                            variant={
                                tag.variant as
                                    | "primary"
                                    | "secondary"
                                    | "link"
                                    | "info"
                                    | "success"
                                    | "warning"
                                    | "danger"
                                    | "accent"
                            }
                        />
                    ))}
                </div>
            </div>
            <WatchValueDisplay sample={watch.sample} />
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

    const [isAddWatchModalOpen, setIsAddWatchModalOpen] = useState(false);

    return (
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing watches...</span>
                    </div>
                </div>
            ) : filteredWatches.length === 0 && showEmptyMessage ? (
                search ? (
                    <div className="flex items-center justify-center h-full text-secondary">
                        no watches match the filter
                    </div>
                ) : (
                    <NoWatchesMessage />
                )
            ) : (
                <div>
                    {filteredWatches.map((watch, index) => (
                        <React.Fragment key={watch.decl.name}>
                            <WatchView watch={watch} model={model} />
                            {/* Add divider after each watch */}
                            <div className="h-px bg-border my-2 w-full"></div>
                        </React.Fragment>
                    ))}

                    {/* Add Watch button - only shown when no search is active */}
                    {!search && (
                        <div className="flex justify-center mt-6">
                            <Tooltip content="Add a new watch">
                                <button
                                    onClick={() => {
                                        setIsAddWatchModalOpen(true);
                                        sendClickEvent("addwatch");
                                    }}
                                    className="flex items-center justify-center h-8 px-3 text-sm text-primary/80 bg-primary/10 border border-primary/20 hover:bg-primary/20 rounded-md cursor-pointer"
                                >
                                    + Add Watch
                                </button>
                            </Tooltip>
                        </div>
                    )}
                </div>
            )}

            <Modal
                isOpen={isAddWatchModalOpen}
                onClose={() => setIsAddWatchModalOpen(false)}
                title="Add Watch"
                className="w-[750px]"
            >
                <NoWatchesMessage hideTitle={true} />
            </Modal>
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
