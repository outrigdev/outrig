import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

const SETTINGS_STORAGE_KEY = "outrig:settings";
const DEFAULT_SHOW_SOURCE = true;

export interface Settings {
    logs?: {
        showSource?: boolean;
        showMilliseconds?: boolean;
        timeFormat?: "absolute" | "relative";
    };
}

const DEFAULT_SHOW_MILLISECONDS = true;
const DEFAULT_TIME_FORMAT = "absolute";

const DEFAULT_SETTINGS: Settings = {
    logs: {
        showSource: DEFAULT_SHOW_SOURCE,
        showMilliseconds: DEFAULT_SHOW_MILLISECONDS,
        timeFormat: DEFAULT_TIME_FORMAT,
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
        return settings.logs?.showSource ?? DEFAULT_SHOW_SOURCE;
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

    logsShowMilliseconds = atom((get) => {
        const settings = get(this.settings);
        return settings.logs?.showMilliseconds ?? DEFAULT_SHOW_MILLISECONDS;
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
        return settings.logs?.timeFormat ?? DEFAULT_TIME_FORMAT;
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
}

const model = new SettingsModel();
export { model as SettingsModel };
