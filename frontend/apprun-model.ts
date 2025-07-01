// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppRunListModel } from "@/apprunlist/apprunlist-model";
import { GoRoutinesModel } from "@/goroutines/goroutines-model";
import { LogViewerModel } from "@/logviewer/logviewer-model";
import { RuntimeStatsModel } from "@/runtimestats/runtimestats-model";
import { WatchesModel } from "@/watches/watches-model";
import { atom, Atom, PrimitiveAtom, getDefaultStore } from "jotai";

type TabModels = {
    logs: LogViewerModel;
    goroutines: GoRoutinesModel;
    watches: WatchesModel;
    runtimestats: RuntimeStatsModel;
};

function createNullModels(): TabModels {
    return { logs: null, goroutines: null, watches: null, runtimestats: null };
}

class AppRunModel {
    // The currently selected app run ID
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");

    // Current model instances - only one set alive at a time
    private currentModels: TabModels = createNullModels();

    // Internal atom that handles model lifecycle based on selectedAppRunId
    private modelsAtom: Atom<TabModels> = atom(
        (get) => {
            const appRunId = get(this.selectedAppRunId);
            
            // Always dispose current models first
            this.disposeCurrentModels();
            
            if (!appRunId) {
                this.currentModels = createNullModels();
                return this.currentModels;
            }

            // Create new models for the new appRunId
            this.currentModels = {
                logs: new LogViewerModel(appRunId),
                goroutines: new GoRoutinesModel(appRunId),
                watches: new WatchesModel(appRunId),
                runtimestats: new RuntimeStatsModel(appRunId)
            };

            return this.currentModels;
        }
    );

    // Derived atom that looks up the app run info from the app run list
    appRunInfoAtom: Atom<AppRunInfo> = atom((get) => {
        const appRunId = get(this.selectedAppRunId);
        if (!appRunId) {
            return null;
        }

        const appRuns = get(AppRunListModel.appRuns);
        return appRuns.find((run: AppRunInfo) => run.apprunid === appRunId) || null;
    });

    // Simple derived atoms that just return the current models
    logsModel: Atom<LogViewerModel> = atom((get) => {
        const models = get(this.modelsAtom);
        return models.logs;
    });

    goRoutinesModel: Atom<GoRoutinesModel> = atom((get) => {
        const models = get(this.modelsAtom);
        return models.goroutines;
    });

    watchesModel: Atom<WatchesModel> = atom((get) => {
        const models = get(this.modelsAtom);
        return models.watches;
    });

    runtimeStatsModel: Atom<RuntimeStatsModel> = atom((get) => {
        const models = get(this.modelsAtom);
        return models.runtimestats;
    });

    // Method to dispose current models
    private disposeCurrentModels() {
        this.currentModels.logs?.dispose();
        this.currentModels.goroutines?.dispose();
        this.currentModels.watches?.dispose();
        this.currentModels.runtimestats?.dispose();
        this.currentModels = createNullModels();
    }

    // Method to dispose all models by clearing the selected app run
    dispose() {
        const store = getDefaultStore();
        store.set(this.selectedAppRunId, "");
    }
}

// Export a singleton instance
const model = new AppRunModel();
export { model as AppRunModel };
