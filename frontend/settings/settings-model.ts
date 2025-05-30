// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, getDefaultStore, PrimitiveAtom } from "jotai";
import { CodeLinkType } from "../codelink/codelink-model";

export interface LogSettings {
    showSource: boolean;
    showTimestamp: boolean;
    showMilliseconds: boolean;
    timeFormat: "absolute" | "relative";
    showLineNumbers: boolean;
    emojiReplacement: "never" | "outrig" | "always";
}

export interface CodeLinkSettings {
    linkType: CodeLinkType;
}

const SETTINGS_STORAGE_KEY = "outrig:settings";
const DEFAULT_SHOW_SOURCE = true;
const DEFAULT_SHOW_LINE_NUMBERS = true;

export interface Settings {
    logs?: Partial<LogSettings>;
    codeLink?: Partial<CodeLinkSettings>;
}

const DEFAULT_SHOW_MILLISECONDS = true;
const DEFAULT_TIME_FORMAT = "absolute";
const DEFAULT_SHOW_TIMESTAMP = true;
const DEFAULT_EMOJI_REPLACEMENT = "outrig";
const DEFAULT_CODE_LINK_TYPE: CodeLinkType = "picker";

const DEFAULT_SETTINGS: Settings = {
    logs: {
        showSource: DEFAULT_SHOW_SOURCE,
        showTimestamp: DEFAULT_SHOW_TIMESTAMP,
        showMilliseconds: DEFAULT_SHOW_MILLISECONDS,
        timeFormat: DEFAULT_TIME_FORMAT,
        showLineNumbers: DEFAULT_SHOW_LINE_NUMBERS,
        emojiReplacement: DEFAULT_EMOJI_REPLACEMENT,
    },
    codeLink: {
        linkType: DEFAULT_CODE_LINK_TYPE,
    },
};

function loadSettings(): Settings | null {
    const storedSettings = localStorage.getItem(SETTINGS_STORAGE_KEY);
    if (!storedSettings) {
        return null;
    }

    try {
        const parsedSettings = JSON.parse(storedSettings);
        return parsedSettings;
    } catch (e) {
        console.error("Failed to parse settings from localStorage:", e);
        return null;
    }
}

function saveSettings(settings: Settings): void {
    localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
}

class SettingsModel {
    settings: PrimitiveAtom<Settings> = atom<Settings>(loadSettings());

    constructor() {}

    logsShowSource = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.showSource ?? DEFAULT_SHOW_SOURCE;
    });

    setLogsShowSource(value: boolean): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                showSource: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    logsShowTimestamp = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.showTimestamp ?? DEFAULT_SHOW_TIMESTAMP;
    });

    setLogsShowTimestamp(value: boolean): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                showTimestamp: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    logsShowMilliseconds = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.showMilliseconds ?? DEFAULT_SHOW_MILLISECONDS;
    });

    setLogsShowMilliseconds(value: boolean): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                showMilliseconds: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    logsTimeFormat = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.timeFormat ?? DEFAULT_TIME_FORMAT;
    });

    setLogsTimeFormat(value: "absolute" | "relative"): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                timeFormat: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    logsShowLineNumbers = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.showLineNumbers ?? DEFAULT_SHOW_LINE_NUMBERS;
    });

    setLogsShowLineNumbers(value: boolean): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                showLineNumbers: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    logsEmojiReplacement = atom((get) => {
        const settings = get(this.settings);
        return settings?.logs?.emojiReplacement ?? DEFAULT_EMOJI_REPLACEMENT;
    });

    setLogsEmojiReplacement(value: "never" | "outrig" | "always"): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            logs: {
                ...(currentSettings.logs || {}),
                emojiReplacement: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }

    // Combined atom for all log settings
    logsSettings = atom<LogSettings>((get) => {
        return {
            showSource: get(this.logsShowSource),
            showTimestamp: get(this.logsShowTimestamp),
            showMilliseconds: get(this.logsShowMilliseconds),
            timeFormat: get(this.logsTimeFormat),
            showLineNumbers: get(this.logsShowLineNumbers),
            emojiReplacement: get(this.logsEmojiReplacement),
        };
    });

    codeLinkType = atom((get) => {
        const settings = get(this.settings);
        return settings?.codeLink?.linkType ?? DEFAULT_CODE_LINK_TYPE;
    });

    setCodeLinkType(value: CodeLinkType): void {
        const currentSettings = getDefaultStore().get(this.settings) || {};
        const newSettings = {
            ...currentSettings,
            codeLink: {
                ...(currentSettings.codeLink || {}),
                linkType: value,
            },
        };
        getDefaultStore().set(this.settings, newSettings);
        saveSettings(newSettings);
    }
}

const model = new SettingsModel();
export { model as SettingsModel };
