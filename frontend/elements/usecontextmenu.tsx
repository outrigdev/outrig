// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

interface ContextMenuItem {
    label: string;
    onClick: () => void;
    disabled?: boolean;
}

interface ContextMenuSeparator {
    type: "separator";
}

type ContextMenuEntry = ContextMenuItem | ContextMenuSeparator;

interface ContextMenuState {
    isOpen: boolean;
    position: { x: number; y: number };
    items: ContextMenuEntry[];
}

export function useContextMenu() {
    const instanceRef = useRef(0);
    const [menuState, setMenuState] = useState<ContextMenuState>({
        isOpen: false,
        position: { x: 0, y: 0 },
        items: [],
    });

    const showContextMenu = useCallback((e: React.MouseEvent, items: ContextMenuEntry[]) => {
        e.preventDefault();
        instanceRef.current = Date.now();
        setMenuState({
            isOpen: true,
            position: { x: e.clientX, y: e.clientY },
            items,
        });
    }, []);

    const hideContextMenu = useCallback(() => {
        setMenuState((prev) => ({ ...prev, isOpen: false }));
    }, []);

    // Click outside to close
    useEffect(() => {
        if (!menuState.isOpen) return;

        const handleClickOutside = (event: MouseEvent) => {
            // Check if the click is inside the context menu
            const target = event.target as Element;
            const contextMenuElement = document.querySelector("[data-context-menu]");
            if (contextMenuElement && contextMenuElement.contains(target)) {
                return; // Don't close if clicking inside the menu
            }
            hideContextMenu();
        };

        // Add listener with small delay to avoid immediate closure
        const timeoutId = setTimeout(() => {
            document.addEventListener("mousedown", handleClickOutside);
        }, 100);

        return () => {
            clearTimeout(timeoutId);
            document.removeEventListener("mousedown", handleClickOutside);
        };
    }, [menuState.isOpen, hideContextMenu]);

    // Escape key to close
    useEffect(() => {
        if (!menuState.isOpen) return;

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                hideContextMenu();
            }
        };

        document.addEventListener("keydown", handleKeyDown);
        return () => document.removeEventListener("keydown", handleKeyDown);
    }, [menuState.isOpen, hideContextMenu]);

    const handleItemClick = useCallback(
        (item: ContextMenuItem) => {
            if (!item.disabled) {
                item.onClick();
                hideContextMenu();
            }
        },
        [hideContextMenu]
    );

    const contextMenu = menuState.isOpen
        ? createPortal(
              <div
                  key={instanceRef.current}
                  data-context-menu
                  style={{
                      position: "fixed",
                      left: menuState.position.x,
                      top: menuState.position.y,
                  }}
                  className="rounded-md shadow min-w-[160px] text-[13px] text-primary bg-[#c7c7c9] z-[1000] pt-[5px] pb-[4px]"
              >
                  {menuState.items.map((item, index) => {
                      if ("type" in item && item.type === "separator") {
                          return <div key={index} className="mx-2 my-1 h-px bg-gray-400" />;
                      }

                      const menuItem = item as ContextMenuItem;
                      return (
                          <div key={index} className="mx-1 mb-[1px]">
                              <button
                                  onClick={(e) => {
                                      e.stopPropagation();
                                      handleItemClick(menuItem);
                                  }}
                                  disabled={menuItem.disabled}
                                  className="font-system w-full px-2 py-0.5 text-left text-white hover:bg-[#0066cc] hover:text-primary disabled:opacity-50 flex items-center gap-1 rounded-md"
                              >
                                  <span>{menuItem.label}</span>
                              </button>
                          </div>
                      );
                  })}
              </div>,
              document.body
          )
        : null;

    return {
        contextMenu,
        showContextMenu,
        hideContextMenu,
    };
}
