Outrig provides real-time debugging for Go programs, similar to Chrome DevTools. It enables quick log searching, goroutine monitoring, variable watching, and runtime hook execution. Integration requires just one line of code.

### Project Structure

- **Frontend**: React app located in `web/`.
- **Client SDK (Go)**: Main library at project root (`outrig.go`) and additional SDK packages in `pkg/`.
- **Server (Go)**: Server code in `server/`, entry point `server/main-server.go`, and server-specific packages in `server/pkg/`.

### Coding Guidelines

- **Comments**:
    - Avoid all redundant comments that merely restate what is obvious from the code itself
    - Do not add comments that simply repeat the component/function name (e.g., don't comment `AppRunListHeader` with "Header component for app run list")
    - Do not add comments that describe what a function does when the function name already clearly indicates its purpose
    - Only add comments for complex logic, non-obvious behavior, or to explain "why" something is done a certain way
- **Go Conventions**:
    - Don't use custom enum types in Go. Instead, use string constants (e.g., `const StatusRunning = "running"` rather than creating a custom type like `type Status string`).
    - Use string constants for status values, packet types, and other string-based enumerations.
- **TypeScript Imports**:
    - Use `@/init` for imports from different parts of the project (configured in `tsconfig.json` as `"@/*": ["web/*"]`).
    - Prefer relative imports (`"./name"`) within the same or child directories.
    - Use named exports exclusively; avoid default exports. It's acceptable to export functions directly (e.g., React Components).
- **JSON Field Naming**: All fields must be lowercase, without underscores.
- In TypeScript we have strict null checks off, so no need to add "| null" to all the types.
- In TypeScript for Jotai atoms, if we want to write, we need to type the atom as a PrimitiveAtom<Type>
- Jotai has a bug with strick null checks off where if you create a null atom, e.g. atom(null) it does not "type" correctly. That's no issue, just cast it to the proper PrimitiveAtom type (no "| null") and it will work fine.

### Styling

- We use tailwind v4 to style. Custom stuff is defined in app.css. We have both light/dark mode styles that are defined via CSS variables.

### Code Generation

- **TypeScript Types**: TypeScript types are automatically generated from Go types. After modifying Go types in `pkg/rpctypes/rpctypes.go`, run `task generate` to update the TypeScript type definitions in `web/types/rpctypes.d.ts`.
- **RPC Client API**: The RPC client API is also generated from Go types. The `task generate` command updates both the TypeScript types and the RPC client API.
- **Manual Edits**: Do not manually edit generated files like `web/types/rpctypes.d.ts` or `web/rpc/rpcclientapi.ts`. Instead, modify the source Go types and run `task generate`.

### Documentation References

- Creating a new RPC API: Refer to `aidocs/newrpcapi.md`
- Creating a new Event: Refer to `aidocs/newevent.md`
- Subscribing to Events on the Frontend: Refer to `aidocs/events.md`
- General RPC documentation: Refer to `aidocs/rpc.md`

### RPC Communication

- Use the Outrig RPC system to communicate between the TypeScript frontend and Go backend. Methods are defined in `rpctypes.go` and exposed through generated code in `rpcclientapi.ts`. Refer to `aidocs/rpc.md` for details on usage and options.
- RPC calls are highly performant, typically running over WebSockets locally on the same machine.
- The RPC system is initialized in `web/init.ts` which creates a global `DefaultRpcClient` that should be used throughout the application. Don't create new RPC clients in components.
- To use the RPC client in a component, import `DefaultRpcClient` from `./init` and set it in the AppModel using `AppModel.setRpcClient(DefaultRpcClient)`.

### Data Structures

- **AppRunPeer**: Represents a connection to a running Go application. Each app run has a unique ID and contains information about the app, logs, and goroutines. For detailed information about AppRunPeer and application lifecycle management, refer to `aidocs/apppeer.md`.
- **CirBuf**: A generic circular buffer implementation used for storing logs and other data. Use the `GetAll()` method to retrieve all items in the buffer.
- **SyncMap**: A thread-safe map implementation. Use the `Keys()` method to get all keys and `GetEx()` to safely retrieve values.

### Frontend Architecture

- The application uses Jotai for state management. The main app state is defined in `web/appmodel.ts`. For detailed information on state management, refer to `aidocs/state-management.md`.
- When working with Jotai atoms that need to be updated, define them as `PrimitiveAtom<Type>` rather than just `atom<Type>`.
- The frontend is organized into components for different views (LogViewer, AppRunList, etc.) that use the AppModel to access shared state.
- The app uses a tab-based navigation system where the selected tab determines which component is displayed.
- To handle keyboard events, use keymodel.ts. Regsiter global keys in registerGlobalKeys() and hook them up to the appropriate handlers.

### Data Flow

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│                 │      │                 │      │                 │
│  Go Application │◄────►│  Outrig Server  │◄────►│  Web Frontend   │
│                 │      │                 │      │                 │
└─────────────────┘      └─────────────────┘      └─────────────────┘
       │                        │                        │
       │                        │                        │
       ▼                        ▼                        ▼
  Sends logs,             Collects and              Displays data
  goroutines,             processes data            and provides
  app info                                          user interface
```

- **Go Application**: Monitored application that sends logs, goroutine information, and app info to the Outrig server.
- **Outrig Server**: Collects and processes data from the monitored application, stores it in appropriate data structures (CirBuf, SyncMap), and makes it available via RPC.
- **Web Frontend**: Retrieves data from the server via RPC calls, manages state with Jotai, and renders the UI components.
