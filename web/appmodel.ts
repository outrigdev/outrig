// AppModel.ts
import { atom, getDefaultStore } from "jotai";

// Create a primitive boolean atom.
class AppModel {
    // UI state
    selectedTab = atom("logs");
    darkMode = atom<boolean>(localStorage.getItem("theme") === "dark");

    constructor() {
        this.applyTheme();
    }

    applyTheme(): void {
        if (localStorage.getItem("theme") === "dark") {
            document.documentElement.dataset.theme = "dark";
        } else {
            document.documentElement.dataset.theme = "light";
        }
    }

    setDarkMode(update: boolean): void {
        if (update) {
            localStorage.setItem("theme", "dark");
        } else {
            localStorage.setItem("theme", "light");
        }
        this.applyTheme();
        getDefaultStore().set(this.darkMode, update);
    }
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
