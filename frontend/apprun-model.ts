// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppRunListModel } from "@/apprunlist/apprunlist-model";
import { atom, Atom, PrimitiveAtom } from "jotai";

class AppRunModel {
    // The currently selected app run ID
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");

    // Derived atom that looks up the app run info from the app run list
    appRunInfoAtom: Atom<AppRunInfo | null> = atom((get) => {
        const appRunId = get(this.selectedAppRunId);
        if (!appRunId) {
            return null;
        }

        const appRuns = get(AppRunListModel.appRuns);
        return appRuns.find((run: AppRunInfo) => run.apprunid === appRunId) || null;
    });
}

// Export a singleton instance
const model = new AppRunModel();
export { model as AppRunModel };
