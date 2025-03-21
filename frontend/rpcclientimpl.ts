import { RpcClient, RpcResponseHelper } from "@/rpc/rpc";
import { RpcRouter } from "@/rpc/rpcrouter";

class RpcClientImpl extends RpcClient {
    constructor(router: RpcRouter, routeId: string) {
        super(router, routeId);
    }

    handle_logstreamupdate(rh: RpcResponseHelper, data: StreamUpdateData) {}
}

export { RpcClientImpl };
