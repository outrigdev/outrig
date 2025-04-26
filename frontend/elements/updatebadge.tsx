// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import { Sparkles } from "lucide-react";
import React from "react";
import { Tooltip } from "./tooltip";

interface UpdateBadgeProps {
    onClick?: () => void;
}

export const UpdateBadge: React.FC<UpdateBadgeProps> = ({ onClick }) => {
    const newerVersion = useAtomValue(AppModel.newerVersion);

    if (!newerVersion) {
        return null;
    }

    return (
        <>
            <div className="mx-1.5 xl:mx-3 h-5 w-[2px] bg-gray-300 dark:bg-gray-600"></div>
            <Tooltip content={`A new version of Outrig is available: ${newerVersion}`}>
                <button
                    onClick={onClick}
                    className="flex items-center gap-2 px-3 h-6 bg-green-700/90 hover:bg-green-600/90 text-white dark:text-primary text-xs rounded-md transition-colors cursor-pointer"
                    aria-label="Update Available"
                >
                    <Sparkles size={16} />
                    <span className="hidden xl:inline">Update Available</span>
                </button>
            </Tooltip>
        </>
    );
};
