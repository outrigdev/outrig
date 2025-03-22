import { LogListInterface, LogPageInterface } from "@/logviewer/logviewer-model";
import { atom, Atom, getDefaultStore, PrimitiveAtom, useAtomValue } from "jotai";
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
    return (
        <div ref={pageRef} className="w-full" style={{ height: defaultItemHeight * totalCount }}>
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
    const { pages, pageSize } = useAtomValue(listAtom);
    return (
        <>
            {pages.map((pageAtom, index) =>
                pageAtom == null ? (
                    <div key={`page-placeholder-${index}`} style={{ height: defaultItemHeight * pageSize }} />
                ) : (
                    <LogPage
                        key={`page-${index}`}
                        pageAtom={pageAtom}
                        defaultItemHeight={defaultItemHeight}
                        lineComponent={lineComponent}
                        pageNum={index}
                        onPageRequired={onPageRequired}
                        vlistRef={vlistRef}
                    />
                )
            )}
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
    // Add a ref to track when we're ignoring the next scroll event
    const ignoreNextScrollRef = useRef<boolean>(false);

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
    }, [version, isPinnedToBottom, vlistRef]);
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
                    // Set flag to ignore the next scroll event
                    ignoreNextScrollRef.current = true;
                    container.scrollTop = maxScrollTop;
                }
            }
        });

        resizeObserver.observe(content);
        return () => resizeObserver.disconnect();
    }, [isPinnedToBottom, vlistRef]);
    return (
        <div
            ref={vlistRef}
            className="w-full overflow-auto"
            style={{ height: containerHeight }}
            onScroll={() => {
                if (!vlistRef.current) return;
                
                // If we should ignore this scroll event, do so and reset the flag
                if (ignoreNextScrollRef.current) {
                    ignoreNextScrollRef.current = false;
                    return;
                }
                
                // Update the follow mode based on scroll position
                const container = vlistRef.current;
                const isAtBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 20;
                
                // Update the pinToBottomAtom if needed
                const store = getDefaultStore();
                if (store.get(pinToBottomAtom) !== isAtBottom) {
                    store.set(pinToBottomAtom, isAtBottom);
                }
            }}
        >
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
