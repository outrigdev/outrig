// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// WebSocket client for Outrig
import { atom, getDefaultStore, PrimitiveAtom } from "jotai";

// Constants
const WarnWebSocketSendSize = 1024 * 1024; // 1MB
const MaxWebSocketSendSize = 5 * 1024 * 1024; // 5MB
const StableConnTime = 2000; // Time after which connection is considered stable
const PingInterval = 5000; // Send ping every 5 seconds

// Event types
const EventType_Rpc = "rpc";
const EventType_Ping = "ping";
const EventType_Pong = "pong";

// Reconnect timeouts in seconds
const ReconnectTimeouts = [1, 1, 2, 5, 10, 10, 30, 60];
const MaxReconnectAttempts = 20;

interface WebSocketOptions {
    url: string;
    onOpen?: () => void;
    onRpcMessage?: (data: RpcMessage) => void;
    onClose?: () => void;
    onError?: (error: Event) => void;
}

export type WSEventType = {
    type: string;
    ts: number;
    data?: any;
};

// Array of reconnect handlers that will be called when connection is reestablished
const reconnectHandlers: (() => void)[] = [];

/**
 * Add a handler that will be called when WebSocket reconnects
 */
export function addWSReconnectHandler(handler: () => void) {
    reconnectHandlers.push(handler);
}

/**
 * Remove a previously added reconnect handler
 */
export function removeWSReconnectHandler(handler: () => void) {
    const index = reconnectHandlers.indexOf(handler);
    if (index > -1) {
        reconnectHandlers.splice(index, 1);
    }
}

export class WebSocketController {
    ws: WebSocket | null = null;
    options: WebSocketOptions;
    reconnectAttempts = 0;
    reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    isConnected = false;
    isOpening = false;
    msgQueue: WSEventType[] = [];
    noReconnect = false;
    onOpenTimeoutId: ReturnType<typeof setTimeout> | null = null;
    pingIntervalId: ReturnType<typeof setInterval> | null = null;

    // Connection state atom
    connectionState: PrimitiveAtom<"connecting" | "connected" | "failed"> = atom<"connecting" | "connected" | "failed">(
        "connecting"
    );

    constructor(options: WebSocketOptions) {
        this.options = options;
        this.connectNow("initial");
        this._startPingInterval();
    }

    _handleWindowFocus() {
        if (this.isConnected) {
            return;
        }
        console.log("[websocket] window focus detected, attempting immediate reconnection");
        this.reconnectAttempts = 0;
        if (this.isOpening) {
            return;
        }
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.connectNow("focus");
    }

    connectNow(desc: string) {
        if (this.isConnected || this.noReconnect) {
            return;
        }
        console.log(`[websocket] trying to connect: ${desc}`);
        this.isOpening = true;
        let url = this.options.url;
        try {
            this.ws = new WebSocket(url);
            this.ws.onopen = (e) => this._onopenHandler(e);
            this.ws.onmessage = (e) => this._onmessageHandler(e);
            this.ws.onclose = (e) => this._oncloseHandler(e);
            this.ws.onerror = (e) => this._onerrorHandler(e);
        } catch (error) {
            console.error("[websocket] Error creating WebSocket:", error);
            this.tryReconnect();
        }
    }

    _onopenHandler(e: Event) {
        console.log("[websocket] connection established");
        this.isConnected = true;
        this.isOpening = false;

        // Update connection state atom
        getDefaultStore().set(this.connectionState, "connected");

        // Set a timeout to reset reconnect attempts if connection stays stable
        this.onOpenTimeoutId = setTimeout(() => {
            this.reconnectAttempts = 0;
            console.log("[websocket] connection stable, reset reconnect counter");
        }, StableConnTime);

        // Call all reconnect handlers
        for (const handler of reconnectHandlers) {
            try {
                handler();
            } catch (err) {
                console.error("[websocket] Error in reconnect handler:", err);
            }
        }

        // Call the onOpen callback if provided
        if (this.options.onOpen) {
            this.options.onOpen();
        }

        // Process any queued messages
        this._runMsgQueue();
    }

    _onmessageHandler(event: MessageEvent) {
        try {
            const data = JSON.parse(event.data) as WSEventType;

            // Handle ping/pong messages for keeping the connection alive
            if (data.type === EventType_Ping) {
                this.sendMessage({ type: EventType_Pong, ts: Date.now() });
                return;
            } else if (data.type === EventType_Pong) {
                // Calculate round-trip time if needed
                if (data.ts) {
                    const now = Date.now();
                    const rtt = now - data.ts;
                    console.debug(`[websocket] RTT: ${rtt}ms`);
                }
                return;
            } else if (data.type === EventType_Rpc) {
                if (this.options.onRpcMessage) {
                    try {
                        this.options.onRpcMessage(data.data);
                    } catch (error) {
                        console.error("[websocket] Error in RPC message handler:", error);
                    }
                }
            } else {
                console.error("[websocket] unknown message type:", data.type);
            }
        } catch (error) {
            console.error("[websocket] Error parsing message:", error);
        }
    }

    _oncloseHandler(event: CloseEvent) {
        // Clear the onOpen timeout if it exists
        if (this.onOpenTimeoutId) {
            clearTimeout(this.onOpenTimeoutId);
            this.onOpenTimeoutId = null;
        }

        if (event.wasClean) {
            console.log("[websocket] connection closed cleanly");
        } else {
            console.log("[websocket] connection closed unexpectedly", event);
        }

        if (this.isConnected || this.isOpening) {
            this.isConnected = false;
            this.isOpening = false;

            // Call the onClose callback if provided
            if (this.options.onClose) {
                this.options.onClose();
            }

            this.tryReconnect();
        }
    }

    _onerrorHandler(error: Event) {
        console.error("[websocket] error:", error);
        if (this.options.onError) {
            this.options.onError(error);
        }
        // No need to call reconnect here as onclose will be called after onerror
    }

    tryReconnect(forceClose = false) {
        if (this.noReconnect) {
            return;
        }

        if (this.isConnected && forceClose) {
            this.ws?.close();
            return;
        }

        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        this.reconnectAttempts++;
        if (this.reconnectAttempts > MaxReconnectAttempts) {
            console.log("[websocket] max reconnect attempts reached, giving up");
            // Update connection state atom to failed
            getDefaultStore().set(this.connectionState, "failed");
            return;
        }

        // Determine timeout based on reconnect attempt count
        let timeout = 60; // Default to 60 seconds
        if (this.reconnectAttempts < ReconnectTimeouts.length) {
            timeout = ReconnectTimeouts[this.reconnectAttempts];
        }

        if (timeout > 0) {
            console.log(`[websocket] will reconnect in ${timeout}s (attempt ${this.reconnectAttempts})`);
        }

        this.reconnectTimer = setTimeout(() => {
            this.connectNow(String(this.reconnectAttempts));
        }, timeout * 1000);
    }

    _startPingInterval() {
        // Clear any existing interval
        if (this.pingIntervalId) {
            clearInterval(this.pingIntervalId);
        }

        // Send ping every 5 seconds to keep the connection alive
        this.pingIntervalId = setInterval(() => {
            if (this.isConnected) {
                this.sendPing();
            }
        }, PingInterval);
    }

    _runMsgQueue() {
        if (!this.isConnected || this.msgQueue.length === 0) {
            return;
        }

        const msg = this.msgQueue.shift();
        this.sendMessage(msg);

        // Process next message after a short delay
        setTimeout(() => {
            this._runMsgQueue();
        }, 100);
    }

    sendPing() {
        const now = Date.now();
        this.sendMessage({ type: EventType_Ping, ts: now });
    }

    /**
     * Send a message to the server
     * @returns boolean indicating if the message was sent successfully
     */
    sendMessage(data: WSEventType): boolean {
        if (!this.isConnected) {
            this.msgQueue.push(data);
            return false;
        }

        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error("[websocket] not connected");
            return false;
        }

        try {
            const message = JSON.stringify(data);

            // Check message size
            const byteSize = new Blob([message]).size;
            if (byteSize > MaxWebSocketSendSize) {
                console.error(`[websocket] message too large (${byteSize} bytes)`);
                return false;
            }

            if (byteSize > WarnWebSocketSendSize) {
                console.warn(`[websocket] large message (${byteSize} bytes)`);
            }

            this.ws.send(message);
            return true;
        } catch (error) {
            console.error("[websocket] Error sending message:", error);
            return false;
        }
    }

    pushRawMessage(data: WSEventType): void {
        if (!this.isConnected) {
            this.msgQueue.push(data);
            return;
        }
        this.sendMessage(data);
    }

    pushRpcMessage(data: RpcMessage): void {
        this.pushRawMessage({ type: EventType_Rpc, ts: Date.now(), data });
    }

    shutdown() {
        this.noReconnect = true;
        this.close();
    }

    close() {
        // Clear all timers
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.onOpenTimeoutId) {
            clearTimeout(this.onOpenTimeoutId);
            this.onOpenTimeoutId = null;
        }

        if (this.pingIntervalId) {
            clearInterval(this.pingIntervalId);
            this.pingIntervalId = null;
        }

        // Close the connection
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }

        this.isConnected = false;
        this.isOpening = false;
    }

    isOpen(): boolean {
        return this.isConnected && this.ws !== null && this.ws.readyState === WebSocket.OPEN;
    }
}
