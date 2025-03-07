import { RpcClient } from "./rpc/rpc";
import { RpcRouter } from "./rpc/rpcrouter";
import { addWSReconnectHandler, WebSocketController } from "./websocket/client";

const WebSocketEndpoint = "ws://localhost:5006/ws";
const RouteIdStorageKey = "outrig:routeid";

let DefaultRouter: RpcRouter = null;
let DefaultRpcClient: RpcClient = null;
let GlobalWS: WebSocketController = null;

class UpstreamWshRpcProxy implements AbstractRpcClient {
    recvRpcMessage(msg: RpcMessage): void {
        if (GlobalWS == null) {
            return;
        }
        GlobalWS.pushRpcMessage(msg);
    }
}

function initRpcSystem() {
    // Check if routeId exists in sessionStorage, otherwise create a new one
    let routeId = sessionStorage.getItem(RouteIdStorageKey);
    if (!routeId) {
        routeId = "frontend:" + crypto.randomUUID();
        sessionStorage.setItem(RouteIdStorageKey, routeId);
    }
    let usp = new URLSearchParams();
    usp.set("routeid", routeId);
    GlobalWS = new WebSocketController({
        url: WebSocketEndpoint + "?" + usp.toString(),
        onRpcMessage: (msg: RpcMessage) => {
            if (DefaultRouter == null) {
                return;
            }
            DefaultRouter.recvRpcMessage(msg);
        },
    });
    window.addEventListener("focus", () => GlobalWS._handleWindowFocus());
    DefaultRouter = new RpcRouter(new UpstreamWshRpcProxy());
    DefaultRpcClient = new RpcClient(DefaultRouter, routeId);
    DefaultRouter.registerRoute(DefaultRpcClient.routeId, DefaultRpcClient);
    addWSReconnectHandler(() => {
        DefaultRouter.reannounceRoutes();
    });
}

export { DefaultRouter, DefaultRpcClient, initRpcSystem };
