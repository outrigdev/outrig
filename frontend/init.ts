// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { registerGlobalKeys } from "@/keymodel";
import { RpcClientImpl } from "@/rpcclientimpl";
import { getDefaultStore } from "jotai";
import { RpcClient } from "./rpc/rpc";
import { RpcRouter } from "./rpc/rpcrouter";
import { addWSReconnectHandler, WebSocketController } from "./websocket/client";

// Make the jotai store available globally
declare global {
    interface Window {
        jotaiStore: any;
    }
}
window.jotaiStore = getDefaultStore();

// Set document title based on development mode
const isDev = import.meta.env.DEV;
if (isDev) {
    document.title = "Outrig (Dev)";
}

// Use the same host and port that served the application
// This ensures WebSocket connections work with tunneled ports or Vite's dev server
const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
const host = window.location.host; // Includes hostname and port
const WebSocketEndpoint = `${protocol}//${host}/ws`;
const RouteIdStorageKey = "outrig:routeid";

let DefaultRouter: RpcRouter = null;
let DefaultRpcClient: RpcClient = null;

class UpstreamWshRpcProxy implements AbstractRpcClient {
    recvRpcMessage(msg: RpcMessage): void {
        const ws = WebSocketController.getInstance();
        if (ws) {
            ws.pushRpcMessage(msg);
        }
    }
}

function initRpcSystem() {
    // Check if routeId exists in sessionStorage, otherwise create a new one
    let routeId = "frontend:" + crypto.randomUUID();
    let usp = new URLSearchParams();
    usp.set("routeid", routeId);
    const wsController = WebSocketController.initialize({
        url: WebSocketEndpoint + "?" + usp.toString(),
        onRpcMessage: (msg: RpcMessage) => {
            if (DefaultRouter == null) {
                return;
            }
            DefaultRouter.recvRpcMessage(msg);
        },
    });
    registerGlobalKeys();
    window.addEventListener("focus", () => wsController._handleWindowFocus());
    DefaultRouter = new RpcRouter(new UpstreamWshRpcProxy());
    DefaultRpcClient = new RpcClientImpl(DefaultRouter, routeId);
    DefaultRouter.registerRoute(DefaultRpcClient.routeId, DefaultRpcClient);
    addWSReconnectHandler(() => {
        DefaultRouter.reannounceRoutes();
    });
}

export { DefaultRouter, DefaultRpcClient, initRpcSystem };
