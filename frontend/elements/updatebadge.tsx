// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { useAtomValue } from "jotai";
import { Download } from "lucide-react";
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
            <div className="mx-3 h-5 w-[2px] bg-gray-300 dark:bg-gray-600"></div>
            <Tooltip content={`A newer version of Outrig is available: ${newerVersion}`}>
                <button
                    onClick={onClick}
                    className="flex items-center p-1 transition-colors cursor-pointer"
                    aria-label="Update Available"
                >
                    <div className="relative">
                        <Download size={16} className="text-accent hover:text-accent-hover" />
                        <div className="absolute -top-1 -right-1 w-2 h-2 bg-accent rounded-full"></div>
                    </div>
                </button>
            </Tooltip>
        </>
    );
};
