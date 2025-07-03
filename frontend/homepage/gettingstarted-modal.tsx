// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { AppModel } from "@/appmodel";
import { Modal } from "@/elements/modal";
import { useAtomValue } from "jotai";
import React from "react";
import { GettingStartedContent } from "./gettingstarted-content";

export const GettingStartedModalContainer: React.FC = () => {
    const isOpen = useAtomValue(AppModel.gettingStartedModalOpen);

    if (!isOpen) {
        return null;
    }

    return (
        <Modal isOpen={isOpen} title="Outrig SDK Integration Instructions" onClose={() => AppModel.closeGettingStartedModal()} className="!w-[700px]">
            <div className="overflow-auto">
                <GettingStartedContent hideTitle={true} hideFooterText={true} />
            </div>
        </Modal>
    );
};
