# Adding a New RPC API to Outrig

Outrig uses a bidirectional RPC system, meaning RPC methods defined in Go are callable from the frontend, and vice versa.

## 1. Define the RPC Method

- RPC methods must end with `Command`.
- The first parameter must always be `ctx context.Context`.
- Optionally, include a second parameter for input data, which can be:
    - A primitive type (`string`, `int`, `bool`, etc.)
    - A struct defined or reused within `rpctypes.go`
- Methods return either `error` or `(result, error)`. Results can similarly be primitive types or structs defined in the file.

## 2. Define Data Types

- Structs for input data or results should be defined or reused within `rpctypes.go`.
- JSON tags must be included for all struct fields and must be lowercase with no capitals or underscores:
    ```go
    Field string `json:"field"`
    ```

## 3. Update Generated Code

- After adding the method and types, regenerate RPC stubs and TypeScript definitions:
    ```shell
    task generate
    ```
- This updates related files like (`rpcclientapi.ts`, `rpcclientapi.go`, `rpctypes.d.ts`).

## 4. Define Streaming Commands

Streaming commands provide continuous, real-time responses from the backend to the frontend, useful for ongoing updates or notifications.

- **Streaming methods** must also end with `Command`.
- The first parameter must always be `ctx context.Context`.
- Optionally, include a second parameter for input data, which must be a struct defined or reused within `rpctypes.go`.
- Streaming methods return a receive-only channel (`<-chan`) of `RespUnion[T]`, where `T` is the expected response struct.

Example definition within `FullRpcInterface`:

```go
StreamTestCommand(ctx context.Context, data SomeData) <-chan RespUnion[SomeResponseType]
```

### Usage Notes

- Define `SomeData` and `SomeResponseType` structs within `rpctypes.go`, including appropriate JSON tags as usual.
- Regenerate RPC stubs and TypeScript definitions after adding streaming commands:

```shell
task generate
```

This ensures frontend and backend communication remains consistent and well-integrated.

### Note

- Outrig RPC is symmetrical; the same file (`rpctypes.go`) handles definitions for both frontend and backend communication.
