// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

declare global {
    type ClientRpcEntry = {
        reqId: string;
        startTs: number;
        command: string;
        msgFn: (msg: RpcMessage) => void;
    };

    interface AbstractRpcClient {
        recvRpcMessage(msg: RpcMessage): void;
    }

    // vdom.WaveKeyboardEvent
    type OutrigKeyboardEvent = {
        type: "keydown" | "keyup" | "keypress" | "unknown";
        key: string;
        code: string;
        repeat?: boolean;
        location?: number;
        shift?: boolean;
        control?: boolean;
        alt?: boolean;
        meta?: boolean;
        cmd?: boolean;
        option?: boolean;
    };

    type KeyPressDecl = {
        mods: {
            Cmd?: boolean;
            Option?: boolean;
            Shift?: boolean;
            Ctrl?: boolean;
            Alt?: boolean;
            Meta?: boolean;
        };
        key: string;
        keyType: string;
    };

    // Add outrigCssLoaded property to Window interface
    interface Window {
        outrigCssLoaded?: boolean;
        jotaiStore: any;
    }
}

export {};
