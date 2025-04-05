import { useAtomValue } from "jotai";
import React, { useEffect, useRef } from "react";
import { AppModel } from "../appmodel";
import { Modal } from "../elements/modal";

// Container component that checks isOpen state
export const SettingsModalContainer: React.FC = () => {
    const isOpen = useAtomValue(AppModel.settingsModalOpen);

    // Only render SettingsModal when isOpen is true
    if (!isOpen) return null;

    return <SettingsModal />;
};

// Actual modal component that doesn't need to check isOpen
export const SettingsModal: React.FC = () => {
    const inputRef = useRef<HTMLInputElement>(null);

    // Focus the hidden input when the component mounts
    useEffect(() => {
        // Force blur on any active element first
        if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
        }

        // Then focus our input
        if (inputRef.current) {
            inputRef.current.focus();
        }
    }, []);

    return (
        <Modal isOpen={true} title="Outrig Settings">
            <div className="text-primary">
                {/* Hidden input to capture focus */}
                <input
                    ref={inputRef}
                    type="text"
                    className="opacity-0 h-0 w-0 absolute"
                    tabIndex={0}
                    aria-hidden="true"
                />

                {/* Settings content will go here */}
                <p>Settings content will be added in the future.</p>
            </div>
        </Modal>
    );
};
