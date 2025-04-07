import { LogListInterface, LogPageInterface } from "@/logviewer/logviewer-model";
import { atom, Atom, PrimitiveAtom, useAtomValue } from "jotai";
import { JSX, useEffect, useLayoutEffect, useRef } from "react";

export interface PageProps {
    pageAtom: Atom<LogPageInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine }) => JSX.Element;
    pageNum: number;
    onPageRequired: (pageNum: number, load: boolean) => void;
    vlistRef: React.RefObject<HTMLDivElement>;
}

function LogPage({ pageAtom, defaultItemHeight, lineComponent, pageNum, onPageRequired, vlistRef }: PageProps) {
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
                return <LineComponent key={line.linenum} line={line} />;
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
            className="w-full"
            data-page={pageNum}
            data-lines={dataLines}
            style={{ height: defaultItemHeight * totalCount }}
        >
            {lineElems}
        </div>
    );
}

export interface LogListProps {
    listAtom: Atom<LogListInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine }) => JSX.Element;
    onPageRequired: (pageNum: number, load: boolean) => void;
    vlistRef: React.RefObject<HTMLDivElement>;
}

function LogList({ listAtom, defaultItemHeight, lineComponent, onPageRequired, vlistRef }: LogListProps) {
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
                        vlistRef={vlistRef}
                    />
                );
            })}
        </>
    );
}

export interface LogVListProps {
    listAtom: Atom<LogListInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine }) => JSX.Element;
    containerHeight: number;
    onPageRequired: (pageNum: number, load: boolean) => void;
    pinToBottomAtom: PrimitiveAtom<boolean>;
    vlistRef: React.RefObject<HTMLDivElement>;
}

export function LogVList({
    listAtom,
    defaultItemHeight,
    lineComponent,
    containerHeight,
    onPageRequired,
    pinToBottomAtom,
    vlistRef,
}: LogVListProps) {
    const contentRef = useRef<HTMLDivElement>(null);
    const isPinnedToBottom = useAtomValue(pinToBottomAtom);
    const versionAtom = useRef(atom((get) => get(listAtom).version)).current;
    const version = useAtomValue(versionAtom);
    const prevVersionRef = useRef<number>(version);
    const { pageSize, trimmedLines } = useAtomValue(listAtom);
    const prevTrimmedLinesRef = useRef<number>(trimmedLines);

    // Handle scroll position adjustment after version changes
    useLayoutEffect(() => {
        const container = vlistRef.current;
        if (!container) return;

        // If version changed, this is a full reset
        if (version !== prevVersionRef.current) {
            // Determine scroll position based on pinToBottom preference
            if (isPinnedToBottom) {
                container.scrollTop = container.scrollHeight;
            } else {
                container.scrollTop = 0;
            }
            prevVersionRef.current = version;
        }
    }, [version, isPinnedToBottom, vlistRef, pageSize]);

    // Handle trimmedLines changes
    useLayoutEffect(() => {
        const container = vlistRef.current;
        if (!container || trimmedLines === prevTrimmedLinesRef.current) return;

        // Calculate how many pages were trimmed
        const trimmedPages = Math.floor(trimmedLines / pageSize);

        if (trimmedPages <= 0) {
            prevTrimmedLinesRef.current = trimmedLines;
            return;
        }

        // Get current scroll position and container dimensions
        const scrollTop = container.scrollTop;

        // Calculate the height of trimmed content
        const trimmedHeight = trimmedPages * pageSize * defaultItemHeight;

        // Case 1: User is viewing content that's been trimmed
        if (scrollTop < trimmedHeight) {
            // Reset to top of available content
            container.scrollTop = 0;
        }
        // Case 2: User is viewing content below the trim point
        else {
            // Adjust scroll position to maintain relative view
            container.scrollTop = scrollTop - trimmedHeight;
        }

        prevTrimmedLinesRef.current = trimmedLines;
    }, [trimmedLines, pageSize, defaultItemHeight, vlistRef]);
    useEffect(() => {
        const content = contentRef.current;
        const container = vlistRef.current;
        if (!content || !container) return;

        const resizeObserver = new ResizeObserver(() => {
            if (isPinnedToBottom) {
                // Calculate the maximum possible scrollTop value
                const maxScrollTop = container.scrollHeight - container.clientHeight;
                // Check if we're already at the bottom (exact comparison)
                if (container.scrollTop !== maxScrollTop) {
                    container.scrollTop = maxScrollTop;
                }
            }
        });

        resizeObserver.observe(content);
        return () => resizeObserver.disconnect();
    }, [isPinnedToBottom, vlistRef]);
    return (
        <div ref={vlistRef} className="w-full overflow-auto" style={{ height: containerHeight }}>
            <div ref={contentRef}>
                <LogList
                    listAtom={listAtom}
                    defaultItemHeight={defaultItemHeight}
                    lineComponent={lineComponent}
                    onPageRequired={onPageRequired}
                    vlistRef={vlistRef}
                />
            </div>
        </div>
    );
}
