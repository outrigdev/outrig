// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

// Maximum number of search history items to store
export const MaxSearchHistoryItems = 30;

/**
 * SearchStore provides persistent search term atoms across tabs
 * This allows search terms to be preserved when switching between tabs
 */
class SearchStoreClass {
    // Simple cache using string keys in format "appName:appRunId:tabName"
    searchTermAtoms: Record<string, PrimitiveAtom<string>> = {};

    /**
     * Create a unique key for the atom cache
     */
    private makeKey(appName: string, appRunId: string, tabName: string): string {
        return `${appName}:${appRunId}:${tabName}`;
    }

    /**
     * Create a unique key for search history in localStorage
     */
    private makeSearchHistoryKey(appName: string, appRunId: string, tabName: string): string {
        const key = this.makeKey(appName, appRunId, tabName);
        return `outrig:searchhistory:${key}`;
    }

    /**
     * Get a search term atom for a specific app, run, and tab
     * Creates a new atom if one doesn't exist, or returns the cached atom
     */
    getSearchTermAtom(appName: string, appRunId: string, tabName: string): PrimitiveAtom<string> {
        // special, so we're only going to key the search terms by appName and tabName (not appRunId)
        const key = this.makeKey(appName, "", tabName);
        // Return existing atom or create a new one
        if (!this.searchTermAtoms[key]) {
            this.searchTermAtoms[key] = atom("");
        }
        return this.searchTermAtoms[key];
    }
    /**
     * Save the current search term to history in localStorage
     */
    saveSearchHistory(appName: string, appRunId: string, tabName: string): void {
        const historyKey = this.makeSearchHistoryKey(appName, appRunId, tabName);
        const searchTermAtom = this.getSearchTermAtom(appName, appRunId, tabName);
        const store = getDefaultStore();

        let searchTerm = store.get(searchTermAtom) || "";
        searchTerm = searchTerm.trim();
        if (!searchTerm) {
            return;
        }
        const existingHistory = this.getSearchHistory(appName, appRunId, tabName);
        if (existingHistory.length > 0 && existingHistory[0] === searchTerm) {
            return;
        }
        const filteredHistory = existingHistory.filter((term) => term !== searchTerm);
        const newHistory = [searchTerm, ...filteredHistory];
        const truncatedHistory = newHistory.slice(0, MaxSearchHistoryItems);
        localStorage.setItem(historyKey, JSON.stringify(truncatedHistory));
    }

    /**
     * Get search history from localStorage
     */
    getSearchHistory(appName: string, appRunId: string, tabName: string): string[] {
        const historyKey = this.makeSearchHistoryKey(appName, appRunId, tabName);
        const historyJson = localStorage.getItem(historyKey);
        if (!historyJson) {
            return [];
        }
        try {
            const history = JSON.parse(historyJson);
            return Array.isArray(history) ? history : [];
        } catch (e) {
            localStorage.removeItem(historyKey);
            return [];
        }
    }

    /**
     * Remove a term from search history
     */
    removeFromSearchHistory(appName: string, appRunId: string, tabName: string, searchTerm: string): void {
        const historyKey = this.makeSearchHistoryKey(appName, appRunId, tabName);
        const existingHistory = this.getSearchHistory(appName, appRunId, tabName);

        // Filter out the term to remove
        const filteredHistory = existingHistory.filter((term) => term !== searchTerm);

        // Save the filtered history back to localStorage
        localStorage.setItem(historyKey, JSON.stringify(filteredHistory));
    }
}

// Create singleton instance
export const SearchStore = new SearchStoreClass();
