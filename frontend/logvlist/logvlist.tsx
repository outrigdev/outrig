// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { LogListInterface, LogPageInterface } from "@/logviewer/logviewer-model";
import { atom, Atom, PrimitiveAtom, useAtom, useAtomValue } from "jotai";
import React, { JSX, useEffect, useLayoutEffect, useRef } from "react";

export interface PageProps {
    pageAtom: Atom<LogPageInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine; pageNum: number; lineIndex: number; onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void }) => JSX.Element;
    pageNum: number;
    onPageRequired: (pageNum: number, load: boolean) => void;
    onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void;
    vlistRef: React.RefObject<HTMLDivElement>;
}

const LogPage = React.memo<PageProps>(
    ({ pageAtom, defaultItemHeight, lineComponent, pageNum, onPageRequired, onContextMenu, vlistRef }) => {
        const { lines, totalCount, loaded } = useAtomValue(pageAtom);
        const LineComponent = lineComponent;
        const pageRef = useRef<HTMLDivElement>(null);

        useEffect(() => {
            if (!pageRef.current) return;

            // Create the observer with options for a buffer
            const options = {
                root: vlistRef.current, // Use the scroll container as the root
                rootMargin: "1000px 0px 1000px 0px", // 1000px buffer above and below viewport
                threshold: 0, // Trigger as soon as any part is visible
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        // When page is visible or about to be visible (within buffer)
                        if (!loaded) {
                            // Request the page data through the callback
                            onPageRequired(pageNum, true);
                        }
                    } else {
                        // When page is no longer visible, we can drop it
                        onPageRequired(pageNum, false);
                    }
                });
            }, options);

            // Start observing the page div
            observer.observe(pageRef.current);

            // Clean up
            return () => observer.disconnect();
        }, [loaded, pageNum, onPageRequired, vlistRef]);

        let lineElems: JSX.Element[] = null;
        if (loaded && lines != null && lines.length > 0) {
            lineElems = lines.map((line, index) => {
                if (line == null) {
                    return <div key={`empty-${index}`} style={{ height: defaultItemHeight }} />;
                } else {
                    return <LineComponent key={line.linenum} line={line} pageNum={pageNum} lineIndex={index} onContextMenu={onContextMenu} />;
                }
            });
        }
        let dataLines = "";
        if (lines != null && lines.length > 0) {
            dataLines = `${lines.length} ${lines[0].linenum}-${lines[lines.length - 1].linenum}`;
        }
        return (
            <div
                ref={pageRef}
                className="w-max min-w-full"
                data-page={pageNum}
                data-lines={dataLines}
                style={{ height: defaultItemHeight * totalCount }}
            >
                {lineElems}
            </div>
        );
    }
);
LogPage.displayName = "LogPage";

export interface LogListProps {
    listAtom: Atom<LogListInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine; pageNum: number; lineIndex: number; onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void }) => JSX.Element;
    onPageRequired: (pageNum: number, load: boolean) => void;
    onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void;
    vlistRef: React.RefObject<HTMLDivElement>;
}

const LogList = React.memo<LogListProps>(({ listAtom, defaultItemHeight, lineComponent, onPageRequired, onContextMenu, vlistRef }) => {
    const { pages, pageSize, trimmedLines } = useAtomValue(listAtom);

    // Calculate how many pages have been trimmed
    const trimmedPages = Math.floor(trimmedLines / pageSize);

    // Use slice to get only the non-trimmed pages
    const visiblePages = pages.slice(trimmedPages);

    return (
        <>
            {visiblePages.map((pageAtom, index) => {
                // Adjust the page number to account for trimmed pages
                const actualPageNum = index + trimmedPages;

                return pageAtom == null ? (
                    <div key={`page-placeholder-${actualPageNum}`} style={{ height: defaultItemHeight * pageSize }} />
                ) : (
                    <LogPage
                        key={`page-${actualPageNum}`}
                        pageAtom={pageAtom}
                        defaultItemHeight={defaultItemHeight}
                        lineComponent={lineComponent}
                        pageNum={actualPageNum}
                        onPageRequired={onPageRequired}
                        onContextMenu={onContextMenu}
                        vlistRef={vlistRef}
                    />
                );
            })}
        </>
    );
});
LogList.displayName = "LogList";

export interface LogVListProps {
    listAtom: Atom<LogListInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine; pageNum: number; lineIndex: number; onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void }) => JSX.Element;
    containerHeight: number;
    onPageRequired: (pageNum: number, load: boolean) => void;
    onContextMenu?: (e: React.MouseEvent, pageNum: number, lineIndex: number) => void;
    pinToBottomAtom: PrimitiveAtom<boolean>;
    vlistRef: React.RefObject<HTMLDivElement>;
}

export const LogVList = React.memo<LogVListProps>(
    ({ listAtom, defaultItemHeight, lineComponent, containerHeight, onPageRequired, onContextMenu, pinToBottomAtom, vlistRef }) => {
        const contentRef = useRef<HTMLDivElement>(null);
        const [isPinnedToBottom, setPinnedToBottom] = useAtom(pinToBottomAtom);
        const versionAtom = useRef(atom((get) => get(listAtom).version)).current;
        const version = useAtomValue(versionAtom);
        const prevVersionRef = useRef<number>(version);
        const { pageSize, trimmedLines } = useAtomValue(listAtom);
        const prevTrimmedLinesRef = useRef<number>(trimmedLines);
        const lastScrollTopRef = useRef<number>(0);
        const lastScrollHeightRef = useRef<number>(0);
        const lastClientHeightRef = useRef<number>(0);

        useEffect(() => {
            const container = vlistRef.current;
            if (!container) return;

            console.log("LogVList useEffect triggered", isPinnedToBottom);
            let raf: number;
            lastScrollTopRef.current = container.scrollTop;
            lastScrollHeightRef.current = container.scrollHeight;
            lastClientHeightRef.current = container.clientHeight;
            const tick = () => {
                const { scrollTop, scrollHeight, clientHeight } = container;

                if (
                    scrollTop < lastScrollTopRef.current && // user scrolled up
                    clientHeight === lastClientHeightRef.current && // not just a resize
                    scrollHeight >= lastScrollHeightRef.current // not a trim
                ) {
                    console.log(
                        "User scrolled up, stopping pinning",
                        scrollTop,
                        lastScrollTopRef.current,
                        clientHeight,
                        lastClientHeightRef.current,
                        scrollHeight,
                        lastScrollHeightRef.current
                    );
                    setPinnedToBottom(false);
                    return;
                }
                if (isPinnedToBottom) {
                    container.scrollTop = scrollHeight - clientHeight;
                }
                lastScrollTopRef.current = container.scrollTop;
                lastScrollHeightRef.current = scrollHeight;
                lastClientHeightRef.current = clientHeight;
                raf = requestAnimationFrame(tick);
            };

            raf = requestAnimationFrame(tick);
            return () => cancelAnimationFrame(raf);
        }, [isPinnedToBottom, setPinnedToBottom, vlistRef]);

        // Handle scroll position when lines are trimmed
        useLayoutEffect(() => {
            const container = vlistRef.current;
            if (!container) return;

            // If trimmedLines increased and we're viewing the trimmed portion
            if (trimmedLines > prevTrimmedLinesRef.current) {
                const trimmedPixels = (trimmedLines - prevTrimmedLinesRef.current) * defaultItemHeight;
                // Only reset scroll if current scroll position is within the trimmed portion
                if (container.scrollTop < trimmedPixels) {
                    // Reset scroll to top
                    container.scrollTop = 0;
                }
                // When outside the trimmed portion, the browser handles the adjustment automatically
            }

            // Update the previous trimmed lines reference
            prevTrimmedLinesRef.current = trimmedLines;
        }, [trimmedLines, defaultItemHeight, vlistRef]);

        return (
            <div ref={vlistRef} className="w-full overflow-auto" style={{ height: containerHeight }}>
                <div ref={contentRef}>
                    <LogList
                        listAtom={listAtom}
                        defaultItemHeight={defaultItemHeight}
                        lineComponent={lineComponent}
                        onPageRequired={onPageRequired}
                        onContextMenu={onContextMenu}
                        vlistRef={vlistRef}
                    />
                </div>
            </div>
        );
    }
);
LogVList.displayName = "LogVList";
