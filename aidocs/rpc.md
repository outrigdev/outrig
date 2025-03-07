# Using the RPC System in TypeScript

This document describes how to use the auto-generated RPC client in TypeScript to call backend methods defined in `rpctypes.go`.

## Import the RPC Client

First, import the default RPC client and generated API:

```typescript
import { DefaultRpcClient } from "@/init";
import { RpcApi } from "./rpcclientapi";
```

## Calling RPC Methods

RPC methods are accessed via the generated `RpcApi` object, using the method names defined in `rpctypes.go`. All methods return a Promise.

Example:

```typescript
RpcApi.UpdateStatusCommand(DefaultRpcClient, { widgetid: "123", status: "active" })
    .then((response) => {
        console.log(response);
    })
    .catch((err) => {
        console.error("RPC error:", err);
    });
```

## RpcOpts

Each RPC call can optionally accept an `RpcOpts` object:

```typescript
type RpcOpts = {
    timeout?: number; // timeout in milliseconds, default is 5000
    noresponse?: boolean; // set true for "fire and forget" requests
    route?: string; // rarely used, for specifying custom route
};
```

Example using `RpcOpts`:

```typescript
RpcApi.UpdateStatusCommand(DefaultRpcClient, { status: "active" }, { timeout: 10000 });

// Fire-and-forget example
RpcApi.LogEventCommand(DefaultRpcClient, { event: "user_login" }, { noresponse: true });
```

## Streaming Methods

Streaming RPC methods return an `AsyncGenerator`:

```typescript
const stream = RpcApi.FileListStreamCommand(DefaultRpcClient, { path: "/my/path" });

for await (const result of stream) {
    console.log(result);
    // To cancel the stream:
    // await stream.next(true);
}
```

The generator yields data of the same type defined in `rpctypes.go`. To cancel the stream, call `next(true)`.

## Notes

To define a new RPC API function, edit `rpctypes.go`. Please refer to the `aidocs/newrpcapi.md` document for more information.
