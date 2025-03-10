// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import * as keyutil from "@/util/keyutil";
import * as jotai from "jotai";

type KeyHandler = (event: OutrigKeyboardEvent) => boolean;

const simpleControlShiftAtom = jotai.atom(false);
const globalKeyMap = new Map<string, (keyEvent: OutrigKeyboardEvent) => boolean>();
const globalChordMap = new Map<string, Map<string, KeyHandler>>();
export const CHORD_TIMEOUT = 2000;

// track current chord state and timeout (for resetting)
let activeChord: string | null = null;
let chordTimeout: NodeJS.Timeout = null;

function resetChord() {
    activeChord = null;
    if (chordTimeout) {
        clearTimeout(chordTimeout);
        chordTimeout = null;
    }
}

function setActiveChord(activeChordArg: string) {
    if (chordTimeout) {
        clearTimeout(chordTimeout);
    }
    activeChord = activeChordArg;
    chordTimeout = setTimeout(() => resetChord(), CHORD_TIMEOUT);
}

let lastHandledEvent: KeyboardEvent | null = null;

// returns [keymatch, T]
function checkKeyMap<T>(keyEvent: OutrigKeyboardEvent, keyMap: Map<string, T>): [string, T] {
    for (const key of keyMap.keys()) {
        console.log("checkKeyPressed", keyEvent, key);
        if (keyutil.checkKeyPressed(keyEvent, key)) {
            const val = keyMap.get(key);
            return [key, val];
        }
    }
    return [null, null];
}

function appHandleKeyDown(keyEvent: OutrigKeyboardEvent): boolean {
    console.log("appHandleKeyDown", keyEvent);
    const nativeEvent = (keyEvent as any).nativeEvent;
    if (lastHandledEvent != null && nativeEvent != null && lastHandledEvent === nativeEvent) {
        return false;
    }
    lastHandledEvent = nativeEvent;
    if (activeChord) {
        console.log("handle activeChord", activeChord);
        // If we're in chord mode, look for the second key.
        const chordBindings = globalChordMap.get(activeChord);
        const [, handler] = checkKeyMap(keyEvent, chordBindings);
        if (handler) {
            resetChord();
            return handler(keyEvent);
        } else {
            // invalid chord; reset state and consume key
            resetChord();
            return true;
        }
    }
    const [chordKeyMatch] = checkKeyMap(keyEvent, globalChordMap);
    if (chordKeyMatch) {
        setActiveChord(chordKeyMatch);
        return true;
    }

    const [, globalHandler] = checkKeyMap(keyEvent, globalKeyMap);
    console.log("globalHandler", globalHandler);
    if (globalHandler) {
        const handled = globalHandler(keyEvent);
        if (handled) {
            return true;
        }
    }
    return false;
}

function registerGlobalKeys() {
    globalKeyMap.set("Ctrl:1", () => {
        AppModel.selectAppRunsTab();
        return true;
    });
    globalKeyMap.set("Ctrl:2", () => {
        AppModel.selectLogsTab();
        return true;
    });
    globalKeyMap.set("Ctrl:3", () => {
        AppModel.selectGoRoutinesTab();
        return true;
    });
}

function getAllGlobalKeyBindings(): string[] {
    const allKeys = Array.from(globalKeyMap.keys());
    return allKeys;
}

// these keyboard events happen *anywhere*, even if you have focus in an input or somewhere else.
function handleGlobalKeyboardEvents(keyEvent: OutrigKeyboardEvent): boolean {
    for (const key of globalKeyMap.keys()) {
        if (keyutil.checkKeyPressed(keyEvent, key)) {
            const handler = globalKeyMap.get(key);
            if (handler == null) {
                return false;
            }
            return handler(keyEvent);
        }
    }
    return false;
}

export { appHandleKeyDown, getAllGlobalKeyBindings, registerGlobalKeys };
