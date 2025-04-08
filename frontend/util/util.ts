import base64 from "base64-js";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function isBlank(str: string | null | undefined): str is null | undefined {
    return str == null || str === "";
}

export function base64ToString(b64: string): string | null {
    if (b64 == null) {
        return null;
    }
    if (b64 == "") {
        return "";
    }
    const stringBytes = base64.toByteArray(b64);
    return new TextDecoder().decode(stringBytes);
}

export function stringToBase64(input: string): string {
    const stringBytes = new TextEncoder().encode(input);
    return base64.fromByteArray(stringBytes);
}

export function base64ToArray(b64: string): Uint8Array {
    const rawStr = atob(b64);
    const rtnArr = new Uint8Array(new ArrayBuffer(rawStr.length));
    for (let i = 0; i < rawStr.length; i++) {
        rtnArr[i] = rawStr.charCodeAt(i);
    }
    return rtnArr;
}

export function boundNumber(num: number, min: number, max: number): number | null {
    if (num == null || typeof num != "number" || isNaN(num)) {
        return null;
    }
    return Math.min(Math.max(num, min), max);
}

export async function consumeGenerator(gen: AsyncGenerator<any, any, any>) {
    let idx = 0;
    try {
        for await (const msg of gen) {
            console.log("gen", idx, msg);
            idx++;
        }
        const result = await gen.return(undefined);
        console.log("gen done", result.value);
    } catch (e) {
        console.log("gen error", e);
    }
}

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

/**
 * Formats a timestamp as a human-readable relative time string (e.g., "just now", "30s ago")
 */
export function formatRelativeTime(timestamp: number): string {
    const now = Date.now();
    const diffInSeconds = Math.floor((now - timestamp) / 1000);

    if (diffInSeconds < 10) {
        return "just now";
    } else if (diffInSeconds < 60) {
        return `${diffInSeconds}s ago`;
    } else if (diffInSeconds < 3600) {
        const minutes = Math.floor(diffInSeconds / 60);
        return `${minutes}m ago`;
    } else if (diffInSeconds < 86400) {
        const hours = Math.floor(diffInSeconds / 3600);
        const minutes = Math.floor((diffInSeconds % 3600) / 60);
        return minutes > 0 ? `${hours}h ${minutes}m ago` : `${hours}h ago`;
    } else {
        const days = Math.floor(diffInSeconds / 86400);
        return `${days}d ago`;
    }
}

/**
 * Formats a duration in seconds as a human-readable string (e.g., "5s", "2m 30s", "1h 15m")
 */
export function formatDuration(durationInSeconds: number): string {
    if (durationInSeconds < 60) {
        return `${durationInSeconds}s`;
    } else if (durationInSeconds < 3600) {
        const minutes = Math.floor(durationInSeconds / 60);
        const seconds = durationInSeconds % 60;
        return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
    } else {
        const hours = Math.floor(durationInSeconds / 3600);
        const minutes = Math.floor((durationInSeconds % 3600) / 60);
        return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
    }
}

/**
 * Efficiently merges two arrays of the same type, replacing items in the first array with matching items from the second array,
 * and adding new items from the second array. Uses a map for O(n) time complexity.
 *
 * @param arr1 The base array
 * @param arr2 The array with updates/new items
 * @param keyFn A function that extracts a key from an item, used to determine which items are the same
 * @returns A new array with the merged items
 */
export function mergeArraysByKey<T, K>(arr1: T[], arr2: T[], keyFn: (item: T) => K): T[] {
    const result = [...arr1];

    // Create a map of keys to indices for quick lookups
    const keyToIndexMap = new Map<K, number>();
    for (let i = 0; i < result.length; i++) {
        const key = keyFn(result[i]);
        keyToIndexMap.set(key, i);
    }

    // Process items from arr2
    for (const item2 of arr2) {
        const key2 = keyFn(item2);
        const existingIndex = keyToIndexMap.get(key2);

        if (existingIndex != null) {
            // Replace existing item with the same key
            result[existingIndex] = item2;
        } else {
            // Add new item
            result.push(item2);
            // Update the map with the new item's index
            keyToIndexMap.set(key2, result.length - 1);
        }
    }

    return result;
}

/**
 * Formats a timestamp as an offset from a start time (e.g., "+50s", "+2m30s", "+5h23m")
 * Rounds down (floors) the difference to the nearest second
 */
export function formatTimeOffset(timestamp: number, startTime: number): string {
    if (!timestamp || !startTime) {
        return "";
    }

    // Calculate difference in seconds, flooring to the nearest second
    const diffInSeconds = Math.floor((timestamp - startTime) / 1000);

    // If the difference is negative or zero, return "+0s"
    if (diffInSeconds <= 0) {
        return "+0s";
    }

    if (diffInSeconds < 60) {
        return `+${diffInSeconds}s`;
    } else if (diffInSeconds < 3600) {
        const minutes = Math.floor(diffInSeconds / 60);
        const seconds = diffInSeconds % 60;
        return seconds > 0 ? `+${minutes}m${seconds}s` : `+${minutes}m`;
    } else {
        const hours = Math.floor(diffInSeconds / 3600);
        const minutes = Math.floor((diffInSeconds % 3600) / 60);
        return minutes > 0 ? `+${hours}h${minutes}m` : `+${hours}h`;
    }
}
