import { RpcClient, RpcResponseHelper } from "@/rpc/rpc";
import { RpcRouter } from "@/rpc/rpcrouter";

// Event name for log stream updates
export const LOG_STREAM_UPDATE_EVENT = "logstreamupdate";

class RpcClientImpl extends RpcClient {
    constructor(router: RpcRouter, routeId: string) {
        super(router, routeId);
    }

    handle_logstreamupdate(rh: RpcResponseHelper, data: StreamUpdateData) {
        // Dispatch a simple custom event with the StreamUpdateData
        document.dispatchEvent(new CustomEvent(LOG_STREAM_UPDATE_EVENT, { detail: data }));
    }
}

export { RpcClientImpl };
