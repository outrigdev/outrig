Outrig provides real-time debugging for Go programs, similar to Chrome DevTools. It enables quick log searching, goroutine monitoring, variable watching, and runtime hook execution. Integration requires just one line of code.

### Project Structure

- **Frontend**: React app located in `web/`.
- **Client SDK (Go)**: Main library at project root (`outrig.go`) and additional SDK packages in `pkg/`.
- **Server (Go)**: Server code in `server/`, entry point `server/main-server.go`, and server-specific packages in `server/pkg/`.

### Coding Guidelines

- **Comments**: Avoid redundant comments (e.g., don't comment `runTask()` with `// runs the task`).
- **TypeScript Imports**:
    - Use `@/init` for imports from different parts of the project (configured in `tsconfig.json` as `"@/*": ["web/*"]`).
    - Prefer relative imports (`"./name"`) within the same or child directories.
    - Use named exports exclusively; avoid default exports. It's acceptable to export functions directly (e.g., React Components).
- **JSON Field Naming**: All fields must be lowercase, without underscores.
- In TypeScript we have strict null checks off, so no need to add "| null" to all the types.
- In TypeScript for Jotai atoms, if we want to write, we need to type the atom as a PrimitiveAtom<Type>
- Jotai has a bug with strick null checks off where if you create a null atom, e.g. atom(null) it does not "type" correctly. That's no issue, just cast it to the proper PrimitiveAtom type (no "| null") and it will work fine.

### Documentation References

- Creating a new RPC API: Refer to `aidocs/newrpcapi.md`
- Creating a new Event: Refer to `aidocs/newevent.md`
- Subscribing to Events on the Frontend: Refer to `aidocs/events.md`
- General RPC documentation: Refer to `aidocs/rpc.md`

### RPC Communication

- Use the Outrig RPC system to communicate between the TypeScript frontend and Go backend. Methods are defined in `rpctypes.go` and exposed through generated code in `rpcclientapi.ts`. Refer to `aidocs/rpc.md` for details on usage and options.
- RPC calls are highly performant, typically running over WebSockets locally on the same machine.
