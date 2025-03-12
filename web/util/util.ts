import base64 from "base64-js";
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

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
