import { useAtomValue } from "jotai";
import React, { useEffect, useRef } from "react";
import { AppModel } from "../appmodel";
import { Dropdown } from "../elements/dropdown";
import { Modal } from "../elements/modal";
import { Toggle } from "../elements/toggle";
import { SettingsModel } from "./settings-model";

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

    const showSource = useAtomValue(SettingsModel.logsShowSource);
    const showTimestamp = useAtomValue(SettingsModel.logsShowTimestamp);
    const showMilliseconds = useAtomValue(SettingsModel.logsShowMilliseconds);
    const timeFormat = useAtomValue(SettingsModel.logsTimeFormat);

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

                <div className="p-4 space-y-6">
                    {/* Logs Section */}
                    <div>
                        <h2 className="text-lg font-semibold mb-3">Logs</h2>
                        <div className="space-y-4">
                            <Toggle
                                id="show-source"
                                checked={showSource}
                                onChange={(checked) => SettingsModel.setLogsShowSource(checked)}
                                label="Show Source"
                            />

                            <Toggle
                                id="show-timestamp"
                                checked={showTimestamp}
                                onChange={(checked) => SettingsModel.setLogsShowTimestamp(checked)}
                                label="Show Timestamp"
                            />

                            <Toggle
                                id="show-milliseconds"
                                checked={showMilliseconds}
                                onChange={(checked) => SettingsModel.setLogsShowMilliseconds(checked)}
                                label="Show Milliseconds"
                            />

                            <Dropdown
                                id="time-format"
                                value={timeFormat}
                                onChange={(value) => SettingsModel.setLogsTimeFormat(value as "absolute" | "relative")}
                                options={[
                                    { value: "absolute", label: "Absolute Time" },
                                    { value: "relative", label: "Relative Time" },
                                ]}
                                label="Time Format"
                            />
                        </div>
                    </div>
                </div>
            </div>
        </Modal>
    );
};
