// AppModel.ts
import { atom, getDefaultStore } from "jotai";
import { addWSReconnectHandler, initWebSocket, OutrigWebSocket } from "./websocket/client";

// WebSocket connection constants
const WS_PORT = 5006;
// Use the current host in production, or localhost in development
const WS_HOST = window.location.hostname === 'localhost' ? 'localhost' : window.location.hostname;
const WS_URL = `ws://${WS_HOST}:${WS_PORT}/ws`;

// Create a primitive boolean atom.
class AppModel {
    // UI state
    selectedTab = atom("logs");
    darkMode = atom<boolean>(localStorage.getItem("theme") === "dark");

    // WebSocket connection state
    wsConnected = atom<boolean>(false);
    wsConnection: OutrigWebSocket | null = null;

    constructor() {
        this.applyTheme();
        this.initWebSocket();
    }

    /**
     * Initialize the WebSocket connection
     */
    initWebSocket(): void {
        console.log("[appmodel] Initializing WebSocket connection to", WS_URL);

        this.wsConnection = initWebSocket({
            url: WS_URL,
            onOpen: () => {
                console.log("[appmodel] WebSocket connection established");
                getDefaultStore().set(this.wsConnected, true);
            },
            onClose: () => {
                console.log("[appmodel] WebSocket connection closed");
                getDefaultStore().set(this.wsConnected, false);
            },
            onMessage: (data) => {
                this.handleWebSocketMessage(data);
            },
            onError: (error) => {
                console.error("[appmodel] WebSocket error:", error);
                getDefaultStore().set(this.wsConnected, false);
            },
        });

        // Add a reconnect handler to refresh data when connection is reestablished
        addWSReconnectHandler(this.handleReconnect.bind(this));
    }

    /**
     * Handle WebSocket reconnection
     */
    private handleReconnect(): void {
        console.log("[appmodel] WebSocket reconnected, refreshing data");
        // Add any data refresh logic here
    }

    /**
     * Handle incoming WebSocket messages
     */
    private handleWebSocketMessage(data: any): void {
        // Process incoming messages
        console.log("[appmodel] Received WebSocket message:", data);

        // Add message handling logic here based on message type
        // For example:
        // if (data.type === 'log') {
        //   // Handle log message
        // }
    }

    /**
     * Send a message through the WebSocket connection
     */
    sendMessage(message: any): boolean {
        if (!this.wsConnection || !this.wsConnection.isOpen()) {
            console.error("[appmodel] Cannot send message: WebSocket not connected");
            return false;
        }

        return this.wsConnection.sendMessage(message);
    }

    /**
     * Apply the current theme
     */
    applyTheme(): void {
        if (localStorage.getItem("theme") === "dark") {
            document.documentElement.dataset.theme = "dark";
        } else {
            document.documentElement.dataset.theme = "light";
        }
    }

    /**
     * Set dark mode state
     */
    setDarkMode(update: boolean): void {
        if (update) {
            localStorage.setItem("theme", "dark");
        } else {
            localStorage.setItem("theme", "light");
        }
        this.applyTheme();
        getDefaultStore().set(this.darkMode, update);
    }
}

// Export a singleton instance
const model = new AppModel();
export { model as AppModel };
