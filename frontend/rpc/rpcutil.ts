// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { RpcRouter } from "./rpcrouter";

async function* rpcResponseGenerator(
    router: RpcRouter,
    openRpcs: Map<string, ClientRpcEntry>,
    command: string,
    reqid: string,
    timeout: number
): AsyncGenerator<any, void, boolean> {
    const msgQueue: RpcMessage[] = [];
    let signalFn: () => void;
    let signalPromise = new Promise<void>((resolve) => (signalFn = resolve));
    let timeoutId: NodeJS.Timeout | null = null;
    if (timeout > 0) {
        timeoutId = setTimeout(() => {
            msgQueue.push({ resid: reqid, error: "EC-TIME: timeout waiting for response" });
            signalFn();
        }, timeout);
    }
    const msgFn = (msg: RpcMessage) => {
        msgQueue.push(msg);
        signalFn();
        // reset signal promise
        signalPromise = new Promise<void>((resolve) => (signalFn = resolve));
    };
    openRpcs.set(reqid, {
        reqId: reqid,
        startTs: Date.now(),
        command: command,
        msgFn: msgFn,
    });
    yield null;
    try {
        while (true) {
            while (msgQueue.length > 0) {
                const msg = msgQueue.shift()!;
                if (msg.error != null) {
                    throw new Error(msg.error);
                }
                if (!msg.cont && msg.data == null) {
                    return;
                }
                const shouldTerminate = yield msg.data;
                if (shouldTerminate) {
                    sendRpcCancel(router, reqid);
                    return;
                }
                if (!msg.cont) {
                    return;
                }
            }
            await signalPromise;
        }
    } finally {
        openRpcs.delete(reqid);
        if (timeoutId != null) {
            clearTimeout(timeoutId);
        }
    }
}

function sendRpcCancel(router: RpcRouter, reqid: string) {
    const rpcMsg: RpcMessage = { reqid: reqid, cancel: true };
    router.recvRpcMessage(rpcMsg);
}

function sendRpcCommand(
    router: RpcRouter,
    openRpcs: Map<string, ClientRpcEntry>,
    msg: RpcMessage
): AsyncGenerator<RpcMessage, void, boolean> {
    router.recvRpcMessage(msg);
    if (msg.reqid == null) {
        return null;
    }
    const rtnGen = rpcResponseGenerator(router, openRpcs, msg.command, msg.reqid, msg.timeout);
    rtnGen.next(); // start the generator (run the initialization/registration logic, throw away the result)
    return rtnGen;
}

export { sendRpcCommand };
