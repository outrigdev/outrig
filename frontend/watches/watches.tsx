import { AutoRefreshButton } from "@/elements/autorefreshbutton";
import { CopyButton } from "@/elements/copybutton";
import { RefreshButton } from "@/elements/refreshbutton";
import { Tag } from "@/elements/tag";
import { Tooltip } from "@/elements/tooltip";
import { useOutrigModel } from "@/util/hooks";
import { checkKeyPressed, keydownWrapper } from "@/util/keyutil";
import { formatRelativeTime } from "@/util/util";
import { useAtom, useAtomValue } from "jotai";
import { Filter } from "lucide-react";
import React, { useEffect, useRef, useState } from "react";
import { WatchesModel } from "./watches-model";

// Constants for watch flags
const WatchFlag_Push = 1;
const WatchFlag_Counter = 2;
const WatchFlag_Atomic = 4;
const WatchFlag_Sync = 8;
const WatchFlag_Func = 16;
const WatchFlag_Hook = 32;
const WatchFlag_Settable = 64;

const PulseFadeTime = 3;

// Individual watch view component
interface WatchViewProps {
    watch: WatchSample;
}

// TimestampDot component to show the last update time with animation
interface TimestampDotProps {
    timestamp: number;
}

const TimestampDot: React.FC<TimestampDotProps> = ({ timestamp }) => {
    // Reference to the dot element for animation reset
    const dotRef = useRef<HTMLDivElement>(null);
    // Store the previous timestamp to detect changes
    const prevTsRef = useRef<number>(timestamp);
    // Use a key to force re-render when timestamp changes
    const [animationKey, setAnimationKey] = useState<number>(0);

    // Calculate how much time has passed since the update (in seconds)
    const timeSinceUpdate = Math.max(0, (Date.now() - timestamp) / 1000);

    // Determine if the timestamp is recent (less than 5 seconds old)
    const isRecent = timeSinceUpdate < PulseFadeTime;

    // Reset animation when timestamp changes
    useEffect(() => {
        if (prevTsRef.current !== timestamp) {
            // Update the key to force a re-render with fresh animation
            setAnimationKey((prev) => prev + 1);
            // Update the previous timestamp
            prevTsRef.current = timestamp;
        }
    }, [timestamp]);

    return (
        <Tooltip
            content={
                <div>
                    Updated: {new Date(timestamp).toLocaleTimeString()}
                    <span className="text-muted ml-1">({formatRelativeTime(timestamp)})</span>
                </div>
            }
        >
            <div
                key={animationKey}
                ref={dotRef}
                className="w-2.5 h-2.5 rounded-full"
                style={{
                    backgroundColor: isRecent ? "var(--watch-dot-active)" : "var(--watch-dot-inactive)",
                    animation: isRecent ? `pulse-fade ${PulseFadeTime}s linear forwards` : undefined,
                    opacity: isRecent ? 1 : 0.7,
                }}
            />
        </Tooltip>
    );
};

const WatchView: React.FC<WatchViewProps> = ({ watch }) => {
    // Format the watch value for display
    const formatValue = (watch: WatchSample) => {
        if (watch.error) {
            return <span className="text-error">{watch.error}</span>;
        }

        if (watch.value == null) {
            return <span className="text-muted">null</span>;
        }

        // Try to parse JSON if it looks like JSON
        if (watch.value.startsWith("{") || watch.value.startsWith("[")) {
            try {
                const parsed = JSON.parse(watch.value);
                return <pre className="text-xs whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>;
            } catch {
                // If it's not valid JSON, just display as is
            }
        }

        return <span>{watch.value}</span>;
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
    const searchRef = useRef<HTMLInputElement>(null);
    const filteredCount = useAtomValue(model.filteredCount);
    const totalCount = useAtomValue(model.totalCount);

    // Focus the search input when the component mounts
    useEffect(() => {
        // Use a small timeout to ensure the input is ready
        const timer = setTimeout(() => {
            searchRef.current?.focus();
        }, 50);
        return () => clearTimeout(timer);
    }, []);

    return (
        <div className="py-1 px-1 border-b border-border">
            <div className="flex items-center justify-between">
                <div className="flex items-center flex-grow">
                    <div className="select-none pr-2 text-muted w-10 text-right font-mono flex justify-end items-center">
                        <Filter
                            size={16}
                            className="text-muted"
                            fill="currentColor"
                            stroke="currentColor"
                            strokeWidth={1}
                        />
                    </div>
                    <input
                        ref={searchRef}
                        type="text"
                        placeholder="Filter watches..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        onKeyDown={keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
                            if (checkKeyPressed(keyEvent, "Escape")) {
                                setSearch("");
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "PageUp")) {
                                model.pageUp();
                                return true;
                            }
                            if (checkKeyPressed(keyEvent, "PageDown")) {
                                model.pageDown();
                                return true;
                            }
                            return false;
                        })}
                        className="w-full bg-transparent text-primary translate-y-px placeholder:text-muted text-sm py-1 pl-0 pr-2
                                border-none ring-0 outline-none focus:outline-none focus:ring-0"
                    />
                </div>

                {/* Search stats */}
                <div className="text-xs text-muted mr-2 select-none">
                    {filteredCount}/{totalCount}
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

    // Set the content ref in the model when it changes
    useEffect(() => {
        model.setContentRef(contentRef);
    }, [model]);

    return (
        <div ref={contentRef} className="w-full h-full overflow-auto flex-1 px-0 py-2">
            {isRefreshing ? (
                <div className="flex items-center justify-center h-full">
                    <div className="flex items-center gap-2 text-primary">
                        <span>Refreshing watches...</span>
                    </div>
                </div>
            ) : filteredWatches.length === 0 ? (
                <div className="flex items-center justify-center h-full text-secondary">
                    {search ? "no watches match the filter" : "no watches found"}
                </div>
            ) : (
                <div>
                    {filteredWatches.map((watch, index) => (
                        <React.Fragment key={watch.name}>
                            <WatchView watch={watch} />
                            {/* Add divider after each watch except the last one */}
                            {index < filteredWatches.length - 1 && (
                                <div
                                    className="h-px bg-border my-2"
                                    style={{ minWidth: "100%", width: "9999px" }}
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
