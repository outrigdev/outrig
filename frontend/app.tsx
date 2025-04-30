// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { ToastContainer } from "@/elements/toast";
import { UpdateModalContainer } from "@/elements/update-modal";
import { HomePage } from "@/homepage/homepage";
import { MainApp } from "@/mainapp/mainapp";
import { SettingsModalContainer } from "@/settings/settings-modal";
import { keydownWrapper } from "@/util/keyutil";
import { useAtom, useAtomValue } from "jotai";
import React, { useEffect } from "react";
import { AppModel } from "./appmodel";
import { appHandleKeyDown } from "./keymodel";

interface AppWrapperProps {
    children: React.ReactNode;
}

function AppWrapper({ children }: AppWrapperProps) {
    const isSettingsModalOpen = useAtomValue(AppModel.settingsModalOpen);
    const [toasts, setToasts] = useAtom(AppModel.toasts);
    const selectedTab = useAtomValue(AppModel.selectedTab);
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    const handleToastClose = (id: string) => {
        AppModel.removeToast(id);
    };

    useEffect(() => {
        AppModel.applyTheme();

        const staticKeyDownHandler = keydownWrapper(appHandleKeyDown);
        document.addEventListener("keydown", staticKeyDownHandler);
        return () => {
            document.removeEventListener("keydown", staticKeyDownHandler);
        };
    }, []);

    // Track URL changes and send them to the backend
    useEffect(() => {
        // Send the URL when the component mounts or when tab/appRunId changes
        AppModel.sendBrowserTabUrl();

        // Listen for popstate events (browser back/forward buttons)
        const handlePopState = () => {
            AppModel.handlePopState();
        };

        // Listen for hashchange events
        const handleHashChange = () => {
            AppModel.sendBrowserTabUrl();
        };

        // Listen for focus/blur events to update the focused state
        const handleFocus = () => {
            AppModel.sendBrowserTabUrl();
        };

        const handleBlur = () => {
            AppModel.sendBrowserTabUrl();
        };

        window.addEventListener("popstate", handlePopState);
        window.addEventListener("hashchange", handleHashChange);
        window.addEventListener("focus", handleFocus);
        window.addEventListener("blur", handleBlur);

        // Clean up event listeners
        return () => {
            window.removeEventListener("popstate", handlePopState);
            window.removeEventListener("hashchange", handleHashChange);
            window.removeEventListener("focus", handleFocus);
            window.removeEventListener("blur", handleBlur);
        };
    }, [selectedAppRunId, selectedTab]); // Re-run when selectedAppRunId or selectedTab changes

    return (
        <>
            <div className="h-screen w-screen flex flex-col bg-panel" inert={isSettingsModalOpen || undefined}>
                {children}
                <ToastContainer toasts={toasts} onClose={handleToastClose} />
            </div>
            <SettingsModalContainer />
            <UpdateModalContainer />
            
            {/* Portal container for highlight overlays */}
            <div id="highlight-overlay-root"></div>
        </>
    );
}

function App() {
    const selectedAppRunId = useAtomValue(AppModel.selectedAppRunId);

    return <AppWrapper>{selectedAppRunId ? <MainApp /> : <HomePage />}</AppWrapper>;
}

export { App };
