import { atom, Atom, getDefaultStore, PrimitiveAtom, useAtomValue } from "jotai";
import { JSX, useEffect, useLayoutEffect, useRef } from "react";

export interface LogPageInterface {
    lines: LogLine[];
    totalCount: number;
    loaded: boolean;
}

export interface PageProps {
    pageAtom: Atom<LogPageInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine }) => JSX.Element;
    pageNum: number;
    onPageRequired: (pageNum: number) => void;
}

function LogPage({ pageAtom, defaultItemHeight, lineComponent, pageNum, onPageRequired }: PageProps) {
    const { lines, totalCount, loaded } = useAtomValue(pageAtom);
    const LineComponent = lineComponent;
    const pageRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!pageRef.current) return;

        // Create the observer with options for a buffer
        const options = {
            rootMargin: "200px 0px", // 200px buffer above and below viewport
            threshold: 0, // Trigger as soon as any part is visible
        };

        const observer = new IntersectionObserver((entries) => {
            entries.forEach((entry) => {
                // When page is visible or about to be visible (within buffer)
                if (entry.isIntersecting && !loaded) {
                    // Request the page data through the callback
                    onPageRequired(pageNum);
                }
            });
        }, options);

        // Start observing the page div
        observer.observe(pageRef.current);

        // Clean up
        return () => observer.disconnect();
    }, [loaded, pageNum, onPageRequired]);

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

export interface LogListInterface {
    totalCount: number;
    pageSize: number;
    pages: PrimitiveAtom<LogPageInterface>[];
    version: number;
}

export interface LogListProps {
    listAtom: Atom<LogListInterface>;
    defaultItemHeight: number;
    lineComponent: (props: { line: LogLine }) => JSX.Element;
    onPageRequired: (pageNum: number) => void;
}

function LogList({ listAtom, defaultItemHeight, lineComponent, onPageRequired }: LogListProps) {
    const { totalCount, pages, pageSize } = useAtomValue(listAtom);
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
    onPageRequired: (pageNum: number) => void;
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

    // Store previous version to detect "reset" events
    const prevVersionRef = useRef<number>(version);

    // Handle scroll position adjustment after version changes
    useLayoutEffect(() => {
        console.log(
            "LogVList version changed",
            version,
            vlistRef.current?.scrollHeight,
            vlistRef.current?.scrollTop,
            "ispinnedToBottom=" + isPinnedToBottom
        );

        const container = vlistRef.current;
        if (!container) return;

        // If version changed, this is a full reset
        if (version !== prevVersionRef.current) {
            // Determine scroll position based on pinToBottom preference
            if (isPinnedToBottom) {
                console.log("scrolling to bottom", container.scrollHeight);
                container.scrollTop = container.scrollHeight;
            } else {
                container.scrollTop = 0;
            }
            prevVersionRef.current = version;
        }
    }, [version, isPinnedToBottom, vlistRef]);

    // The resize observer can handle incremental updates
    useEffect(() => {
        const content = contentRef.current;
        const container = vlistRef.current;
        if (!content || !container) return;

        const resizeObserver = new ResizeObserver(() => {
            if (isPinnedToBottom) {
                container.scrollTop = container.scrollHeight;
            }
        });

        resizeObserver.observe(content);
        return () => resizeObserver.disconnect();
    }, [isPinnedToBottom, vlistRef]);

    // Rest of your component...

    return (
        <div
            ref={vlistRef}
            className="w-full overflow-auto"
            style={{ height: containerHeight }}
            onScroll={() => {
                console.log("scrolling", vlistRef.current?.scrollHeight, vlistRef.current?.scrollTop);
                
                // Update the follow mode based on scroll position
                if (vlistRef.current) {
                    const container = vlistRef.current;
                    const isAtBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 20;
                    
                    // Update the pinToBottomAtom if needed
                    const store = getDefaultStore();
                    if (store.get(pinToBottomAtom) !== isAtBottom) {
                        store.set(pinToBottomAtom, isAtBottom);
                    }
                }
            }}
        >
            <div ref={contentRef}>
                <LogList
                    listAtom={listAtom}
                    defaultItemHeight={defaultItemHeight}
                    lineComponent={lineComponent}
                    onPageRequired={onPageRequired}
                />
            </div>
        </div>
    );
}
