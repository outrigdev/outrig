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
import { Pin } from "lucide-react";
import React, { useEffect, useRef, useState } from "react";
import { NoWatchesMessage } from "./nowatchmessage";
import { WatchVal } from "./watch-val";
import { WatchesModel } from "./watches-model";

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
            <div className="text-sm text-primary pb-2">{formatValue()}</div>
            {(sample.len != null || sample.cap != null) && (
                <div className="pb-2 flex gap-3">
                    {sample.len != null && <span className="text-xs text-muted">Length: {sample.len}</span>}
                    {sample.cap != null && <span className="text-xs text-muted">Capacity: {sample.cap}</span>}
                </div>
            )}
        </>
    );
};

const WatchView: React.FC<WatchViewProps> = ({ watch, model }) => {
    const isPinned = useAtomValue(model.getWatchPinnedAtom(watch.decl.name));

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

    const handlePinClick = () => {
        model.toggleWatchPin(watch.decl.name);
    };

    return (
        <div className="pl-4 pr-2 relative">
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
                    {/* Pin button */}
                    <Tooltip content={isPinned ? "Unpin watch" : "Pin watch"}>
                        <button
                            onClick={handlePinClick}
                            className={`flex items-center gap-1 px-2 py-1 rounded text-xs cursor-pointer border transition-colors ${
                                isPinned
                                    ? "text-warning bg-warning/10 border-warning/30"
                                    : "text-secondary hover:text-primary border-border hover:border-primary/30 hover:bg-buttonhover"
                            }`}
                        >
                            <Pin size={14} />
                            {isPinned && <span>Pinned</span>}
                        </button>
                    </Tooltip>
                </div>
            </div>
            <WatchValueDisplay sample={watch.sample} />
            {watch.sample.polldur != null && watch.sample.polldur > 2000 && (
                <div className="absolute bottom-2 right-2 text-xs text-warning/80">
                    Long poll duration: {(watch.sample.polldur / 1000).toFixed(2)}ms
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
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2 pb-14">
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
                </div>
            )}
        </div>
    );
};

// Main watches component that composes the sub-components
interface WatchesProps {
    appRunId: string;
}

// Floating Add Watch button component
interface AddWatchButtonProps {
    model: WatchesModel;
}

const AddWatchButton: React.FC<AddWatchButtonProps> = ({ model }) => {
    const [isAddWatchModalOpen, setIsAddWatchModalOpen] = useState(false);
    const search = useAtomValue(model.searchTerm);

    if (search) {
        return null;
    }

    return (
        <>
            <div className="fixed bottom-10 right-4 z-10">
                <Tooltip content="Add a new watch">
                    <button
                        onClick={() => {
                            setIsAddWatchModalOpen(true);
                            sendClickEvent("addwatch");
                        }}
                        className="flex items-center justify-center h-8 px-3 text-sm text-primary bg-gray-200 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 hover:bg-gray-300 dark:hover:bg-gray-600 rounded-md cursor-pointer shadow-shadow shadow-lg"
                    >
                        + Add Watch
                    </button>
                </Tooltip>
            </div>

            <Modal
                isOpen={isAddWatchModalOpen}
                onClose={() => setIsAddWatchModalOpen(false)}
                title="Add Watch"
                className="w-[800px]"
            >
                <NoWatchesMessage hideTitle={true} />
            </Modal>
        </>
    );
};

export const Watches: React.FC<WatchesProps> = ({ appRunId }) => {
    const model = useOutrigModel(WatchesModel, appRunId);

    if (!model) {
        return null;
    }

    return (
        <div className="w-full h-full flex flex-col relative">
            <WatchesFilters model={model} />
            <WatchesContent model={model} />
            <AddWatchButton model={model} />
        </div>
    );
};
