// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React from "react";
import { Settings } from "lucide-react";
import { Tooltip } from "./tooltip";

interface SettingsButtonProps {
    onClick: () => void;
}

export const SettingsButton: React.FC<SettingsButtonProps> = ({ onClick }) => {
    return (
        <Tooltip content="Settings">
            <button
                onClick={onClick}
                className="flex items-center p-1 pr-2 transition-colors cursor-pointer"
                aria-label="Settings"
            >
                <Settings size={16} className="text-muted hover:text-primary" />
            </button>
        </Tooltip>
    );
};