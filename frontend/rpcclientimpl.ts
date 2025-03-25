import { RpcClient, RpcResponseHelper } from "@/rpc/rpc";
import { RpcRouter } from "@/rpc/rpcrouter";
import { emitter } from "@/events";

class RpcClientImpl extends RpcClient {
    constructor(router: RpcRouter, routeId: string) {
        super(router, routeId);
    }

    handle_logstreamupdate(rh: RpcResponseHelper, data: StreamUpdateData) {
        // Emit the event using mitt
        emitter.emit('logstreamupdate', data);
    }
}

export { RpcClientImpl };
