import { RpcClient } from "./rpc/rpc";
import { RpcRouter } from "./rpc/rpcrouter";
import { addWSReconnectHandler, WebSocketController } from "./websocket/client";

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
    GlobalWS = new WebSocketController({
        url: "ws://localhost:5006",
        onRpcMessage: (msg: RpcMessage) => {
            if (DefaultRouter == null) {
                return;
            }
            DefaultRouter.recvRpcMessage(msg);
        },
    });
    window.addEventListener("focus", () => GlobalWS._handleWindowFocus());
    DefaultRouter = new RpcRouter(new UpstreamWshRpcProxy());
    DefaultRpcClient = new RpcClient(DefaultRouter, "frontend:" + crypto.randomUUID());
    DefaultRouter.registerRoute(DefaultRpcClient.routeId, DefaultRpcClient);
    addWSReconnectHandler(() => {
        DefaultRouter.reannounceRoutes();
    });
}

export { DefaultRouter, DefaultRpcClient, initRpcSystem };
