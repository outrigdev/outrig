// WebSocket client for Outrig

// Constants
const WarnWebSocketSendSize = 1024 * 1024; // 1MB
const MaxWebSocketSendSize = 5 * 1024 * 1024; // 5MB
const StableConnTime = 2000; // Time after which connection is considered stable
const PingInterval = 5000; // Send ping every 5 seconds

// Reconnect timeouts in seconds
const ReconnectTimeouts = [0, 0, 2, 5, 10, 10, 30, 60];

interface WebSocketOptions {
    url: string;
    onOpen?: () => void;
    onMessage?: (data: any) => void;
    onClose?: () => void;
    onError?: (error: Event) => void;
    autoReconnect?: boolean;
    reconnectInterval?: number;
    maxReconnectAttempts?: number;
    tabId?: string; // Optional identifier for this connection
}

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

export class OutrigWebSocket {
    private ws: WebSocket | null = null;
    private options: WebSocketOptions;
    private reconnectAttempts = 0;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private lastPingTime = 0;
    private lastReconnectTime = 0;
    private isConnected = false;
    private isOpening = false;
    private msgQueue: any[] = [];
    private noReconnect = false;
    private onOpenTimeoutId: ReturnType<typeof setTimeout> | null = null;
    private pingIntervalId: ReturnType<typeof setInterval> | null = null;

    constructor(options: WebSocketOptions) {
        this.options = {
            autoReconnect: true,
            reconnectInterval: 1000,
            maxReconnectAttempts: 20,
            tabId: crypto.randomUUID(), // Generate a random ID if not provided
            ...options,
        };
        this.connectNow("initial");
        this.startPingInterval();
    }

    /**
     * Attempt to connect to the WebSocket server
     */
    private connectNow(desc: string) {
        if (this.isConnected || this.noReconnect) {
            return;
        }

        this.lastReconnectTime = Date.now();
        console.log(`[websocket] trying to connect: ${desc}`);

        this.isOpening = true;

        // Construct URL with tabId if provided
        let url = this.options.url;
        if (this.options.tabId) {
            const separator = url.includes("?") ? "&" : "?";
            url = `${url}${separator}tabid=${this.options.tabId}`;
        }

        try {
            this.ws = new WebSocket(url);

            this.ws.onopen = (e) => this.onopen(e);
            this.ws.onmessage = (e) => this.onmessage(e);
            this.ws.onclose = (e) => this.onclose(e);
            this.ws.onerror = (e) => this.onerror(e);
        } catch (error) {
            console.error("[websocket] Error creating WebSocket:", error);
            this.reconnect();
        }
    }

    /**
     * Handle WebSocket open event
     */
    private onopen(e: Event) {
        console.log("[websocket] connection established");
        this.isConnected = true;
        this.isOpening = false;

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
        this.runMsgQueue();
    }

    /**
     * Handle WebSocket message event
     */
    private onmessage(event: MessageEvent) {
        try {
            const data = JSON.parse(event.data) as { type?: string; stime?: number };

            // Handle ping/pong messages for keeping the connection alive
            if (data.type === "ping") {
                this.sendMessage({ type: "pong", stime: Date.now() });
                return;
            } else if (data.type === "pong") {
                // Calculate round-trip time if needed
                if (data.stime) {
                    const now = Date.now();
                    const rtt = now - data.stime;
                    console.debug(`[websocket] RTT: ${rtt}ms`);
                }
                return;
            }

            // Handle regular messages
            if (this.options.onMessage) {
                try {
                    this.options.onMessage(data);
                } catch (error) {
                    console.error("[websocket] Error in message handler:", error);
                }
            }
        } catch (error) {
            console.error("[websocket] Error parsing message:", error);
        }
    }

    /**
     * Handle WebSocket close event
     */
    private onclose(event: CloseEvent) {
        // Clear the onOpen timeout if it exists
        if (this.onOpenTimeoutId) {
            clearTimeout(this.onOpenTimeoutId);
            this.onOpenTimeoutId = null;
        }

        if (event.wasClean) {
            console.log("[websocket] connection closed cleanly");
        } else {
            console.log("[websocket] connection closed unexpectedly");
        }

        if (this.isConnected || this.isOpening) {
            this.isConnected = false;
            this.isOpening = false;

            // Call the onClose callback if provided
            if (this.options.onClose) {
                this.options.onClose();
            }

            this.reconnect();
        }
    }

    /**
     * Handle WebSocket error event
     */
    private onerror(error: Event) {
        console.error("[websocket] error:", error);
        if (this.options.onError) {
            this.options.onError(error);
        }
        // No need to call reconnect here as onclose will be called after onerror
    }

    /**
     * Schedule a reconnection attempt
     */
    private reconnect(forceClose = false) {
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
        if (this.reconnectAttempts > (this.options.maxReconnectAttempts || 20)) {
            console.log("[websocket] max reconnect attempts reached, giving up");
            return;
        }

        // Determine timeout based on reconnect attempt count
        let timeout = 60; // Default to 60 seconds
        if (this.reconnectAttempts < ReconnectTimeouts.length) {
            timeout = ReconnectTimeouts[this.reconnectAttempts];
        }

        // If we just tried to reconnect, use a short timeout
        if (Date.now() - this.lastReconnectTime < 500) {
            timeout = 1;
        }

        if (timeout > 0) {
            console.log(`[websocket] will reconnect in ${timeout}s (attempt ${this.reconnectAttempts})`);
        }

        this.reconnectTimer = setTimeout(() => {
            this.connectNow(String(this.reconnectAttempts));
        }, timeout * 1000);
    }

    /**
     * Start the ping interval
     */
    private startPingInterval() {
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

    /**
     * Process the message queue
     */
    private runMsgQueue() {
        if (!this.isConnected || this.msgQueue.length === 0) {
            return;
        }

        const msg = this.msgQueue.shift();
        this.sendMessage(msg);

        // Process next message after a short delay
        setTimeout(() => {
            this.runMsgQueue();
        }, 100);
    }

    /**
     * Send a ping message to the server
     */
    public sendPing() {
        this.lastPingTime = Date.now();
        this.sendMessage({ type: "ping", stime: this.lastPingTime });
    }

    /**
     * Send a message to the server
     * @returns boolean indicating if the message was sent successfully
     */
    public sendMessage(data: any): boolean {
        if (!this.isConnected) {
            this.msgQueue.push(data);
            return false;
        }

        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error("[websocket] not connected");
            return false;
        }

        try {
            const message = typeof data === "string" ? data : JSON.stringify(data);

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

    /**
     * Push a message to be sent, queuing it if not connected
     */
    public pushMessage(data: any): void {
        if (!this.isConnected) {
            this.msgQueue.push(data);
            return;
        }

        this.sendMessage(data);
    }

    /**
     * Shutdown the WebSocket connection and prevent reconnection
     */
    public shutdown() {
        this.noReconnect = true;
        this.close();
    }

    /**
     * Close the WebSocket connection
     */
    public close() {
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

    /**
     * Check if the WebSocket is currently connected
     */
    public isOpen(): boolean {
        return this.isConnected && this.ws !== null && this.ws.readyState === WebSocket.OPEN;
    }
}

// Global WebSocket instance
let globalWS: OutrigWebSocket | null = null;

/**
 * Initialize the global WebSocket connection
 */
export function initWebSocket(options: WebSocketOptions): OutrigWebSocket {
    if (globalWS) {
        globalWS.shutdown();
    }

    globalWS = new OutrigWebSocket(options);
    return globalWS;
}

/**
 * Get the global WebSocket instance
 */
export function getWebSocket(): OutrigWebSocket | null {
    return globalWS;
}

/**
 * Send a message using the global WebSocket
 */
export function sendWSMessage(data: any): boolean {
    if (!globalWS) {
        console.error("[websocket] No global WebSocket instance");
        return false;
    }

    return globalWS.pushMessage(data) !== undefined;
}
