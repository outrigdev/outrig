declare global {
    type ClientRpcEntry = {
        reqId: string;
        startTs: number;
        command: string;
        msgFn: (msg: RpcMessage) => void;
    };

    interface AbstractRpcClient {
        recvRpcMessage(msg: RpcMessage): void;
    }
}

export {};
