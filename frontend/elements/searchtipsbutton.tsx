// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { SearchTipsPopup } from "@/elements/search-tips-popup";
import { emitter } from "@/events";
import { useAtom } from "jotai";
import { Lightbulb } from "lucide-react";
import React, { useRef } from "react";

interface SearchTipsButtonProps {
    className?: string;
}

export const SearchTipsButton: React.FC<SearchTipsButtonProps> = ({ className }) => {
    const [isSearchTipsOpen, setIsSearchTipsOpen] = useAtom(AppModel.isSearchTipsOpen);
    const searchTipsButtonRef = useRef<HTMLButtonElement>(null);

    const handleButtonClick = () => {
        const newState = !isSearchTipsOpen;
        if (newState) {
            AppModel.openSearchTips();
            
            // If opening the search tips, focus the search input after a short delay
            setTimeout(() => {
                emitter.emit("focussearch");
            }, 50);
        } else {
            AppModel.closeSearchTips();
        }
    };

    const handleClose = () => {
        AppModel.closeSearchTips();
        
        // Focus the search input after closing the popup
        setTimeout(() => {
            emitter.emit("focussearch");
        }, 50);
    };

    return (
        <>
            <button
                ref={searchTipsButtonRef}
                onClick={handleButtonClick}
                className={`p-1 rounded cursor-pointer transition-colors ${
                    isSearchTipsOpen
                        ? "bg-primary/20 text-primary hover:bg-primary/30"
                        : "text-muted hover:bg-buttonhover hover:text-primary"
                } ${className || ""}`}
                aria-pressed={isSearchTipsOpen}
                aria-label="Search tips"
            >
                <Lightbulb size={16} />
            </button>

            {/* Search tips popup */}
            <SearchTipsPopup
                referenceElement={searchTipsButtonRef.current}
                isOpen={isSearchTipsOpen}
                onClose={handleClose}
            />
        </>
    );
};

SearchTipsButton.displayName = "SearchTipsButton";