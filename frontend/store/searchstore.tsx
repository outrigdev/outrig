// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atom, PrimitiveAtom } from "jotai";

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
}

// Create singleton instance
export const SearchStore = new SearchStoreClass();
