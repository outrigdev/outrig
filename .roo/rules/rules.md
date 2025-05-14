Outrig provides real-time debugging for Go programs, similar to Chrome DevTools. It enables quick log searching, goroutine monitoring, variable watching, and runtime hook execution. Integration requires just one line of code.

### Project Structure

- **Frontend**: React app located in `frontend/`.
- **Client SDK (Go)**: Main library at project root (`outrig.go`) and additional SDK packages in `pkg/`. Data structures are in ds.go. Main coordination happens in controller.go. Various stats are collected by the collectors in pkg/collector/\*
- **Server (Go)**: Server code in `server/`, entry point `server/main-server.go`, and server-specific packages in `server/pkg/`.

### Coding Guidelines

- **Go Conventions**:
    - Don't use custom enum types in Go. Instead, use string constants (e.g., `const StatusRunning = "running"` rather than creating a custom type like `type Status string`).
    - Use string constants for status values, packet types, and other string-based enumerations.
    - in Go code, prefer using Printf() vs Println()
    - use "Make" as opposed to "New" for struct initialization func names
    - in general const decls go at the top fo the file (before types and functions)
- **TypeScript Imports**:
    - Use `@/...` for imports from different parts of the project (configured in `tsconfig.json` as `"@/*": ["frontend/*"]`).
    - Prefer relative imports (`"./name"`) only within the same directory.
    - Use named exports exclusively; avoid default exports. It's acceptable to export functions directly (e.g., React Components).
    - Our indent is 4 spaces
- **JSON Field Naming**: All fields must be lowercase, without underscores.
- **TypeScript Conventions**
    - **Type Handling**:
        - In TypeScript we have strict null checks off, so no need to add "| null" to all the types.
        - In TypeScript for Jotai atoms, if we want to write, we need to type the atom as a PrimitiveAtom<Type>
        - Jotai has a bug with strict null checks off where if you create a null atom, e.g. atom(null) it does not "type" correctly. That's no issue, just cast it to the proper PrimitiveAtom type (no "| null") and it will work fine.
        - Generally never use "=== undefined" or "!== undefined". This is bad style. Just use a "== null" or "!= null" unless it is a very specific case where we need to distinguish undefined from null.
    - **Coding Style**:
        - Use all lowercase filenames (except where case is actually important like Taskfile.yml)
        - Import the "cn" function from "@/util/util" to do classname / clsx class merge (it uses twMerge underneath)
        - For element variants use class-variance-authority
    - **Component Practices**:
        - Make sure to add cursor-pointer to buttons/links and clickable items
        - useAtom() and useAtomValue() are react HOOKS, so they must be called at the component level not inline in JSX
        - If you use React.memo(), make sure to add a displayName for the component

### Styling

- We use tailwind v4 to style. Custom stuff is defined in app.css. We have both light/dark mode styles that are defined via CSS variables. Note this means there is not a tailwind.config.ts file! Tailwind v4 uses CSS variables (defined in app.css) to produce custom tailwind classes and overrides.
- The app must support both light and dark modes. We prefer to use the same tailwind clases for both light/dark and define overrides in the light/dark sections in app.css. So text-primary is a black color in light mode, but a white color in dark mode.
- _never_ use cursor-help (it looks terrible)

### Code Generation

- **TypeScript Types**: TypeScript types are automatically generated from Go types. After modifying Go types in `pkg/rpctypes/rpctypes.go`, run `task generate` to update the TypeScript type definitions in `frontend/types/rpctypes.d.ts`.
- **RPC Client API**: The RPC client API is also generated from Go types. The `task generate` command updates both the TypeScript types and the RPC client API.
- **Manual Edits**: Do not manually edit generated files like `frontend/types/rpctypes.d.ts` or `frontend/rpc/rpcclientapi.ts`. Instead, modify the source Go types and run `task generate`.

### Documentation References

- Creating a new RPC API: Refer to `aidocs/newrpcapi.md`
- Creating a new Event: Refer to `aidocs/newevent.md`
- Subscribing to Events on the Frontend: Refer to `aidocs/events.md`
- Backend RPC Events System: Refer to `aidocs/backendrpsevents.md`
- General RPC documentation: Refer to `aidocs/rpc.md`
- Keyboard event handling: Refer to `aidocs/keyboardevents.md`

### RPC Communication

- Use the Outrig RPC system to communicate between the TypeScript frontend and Go backend. Methods are defined in `rpctypes.go` and exposed through generated code in `rpcclientapi.ts`. Refer to `aidocs/rpc.md` for details on usage and options.
- RPC calls are highly performant, typically running over WebSockets locally on the same machine.
- The RPC system is initialized in `frontend/init.ts` which creates a global `DefaultRpcClient` that should be used throughout the application. Don't create new RPC clients in components.
- To use the RPC client in a component, import `DefaultRpcClient` from `./init` and set it in the AppModel using `AppModel.setRpcClient(DefaultRpcClient)`.

### Data Structures

- **AppRunPeer**: Represents a connection to a running Go application. Each app run has a unique ID and contains information about the app, logs, and goroutines. For detailed information about AppRunPeer and application lifecycle management, refer to `aidocs/apppeer.md`.
- **CirBuf**: A generic circular buffer implementation used for storing logs and other data. Use the `GetAll()` method to retrieve all items in the buffer.
- **SyncMap**: A thread-safe map implementation. Use the `Keys()` method to get all keys and `GetEx()` to safely retrieve values.

### Frontend Architecture

- The application uses Jotai for state management. The main app state is defined in `frontend/appmodel.ts`. For detailed information on state management, refer to `aidocs/state-management.md`.
- When working with Jotai atoms that need to be updated, define them as `PrimitiveAtom<Type>` rather than just `atom<Type>`.
- The frontend is organized into components for different views (LogViewer, AppRunList, etc.) that use the AppModel to access shared state.
- The app uses a tab-based navigation system where the selected tab determines which component is displayed.
- To handle keyboard events, use keymodel.ts. Register global keys in registerGlobalKeys() and hook them up to the appropriate handlers.
- New modal containers should be added to app.tsx (not mainapp.tsx)

### Data Flow

- **Go Application**: Monitored application that sends logs, goroutine information, and app info to the Outrig server through packets sent via the controller (SendPacket).
- **Outrig Server**: Collects and processes data from the monitored application, stores it in appropriate data structures (CirBuf, SyncMap), and makes it available via RPC. Each connected SDK client creates an AppRunPeer. Go is more efficient at holding and scanning data than JavaScript so we prefer to do storage and processing on the server.
- **Web Frontend**: React application that communicates with the server via WebSocket using RPC calls (not raw packets). Retrieves data from the server, manages state with Jotai, and renders the UI components.
- Normally the web frontend and server run on the same host (localhost) so communication is very fast with near-zero latency.

### Notes

- **CRITICAL: Completion format MUST be: "Done: [one-line description]"**
- **Keep your Task Completed summaries VERY short**
- **No lengthy pre-completion summaries** - Do not provide detailed explanations of implementation before using attempt_completion
- **No recaps of changes** - Skip explaining what was done before completion
- **Go directly to completion** - After making changes, proceed directly to attempt_completion without summarizing
- The project is currently an un-released POC / MVP. Do not worry about backward compatibility when making changes
- With React hooks, always complete all hook calls at the top level before any conditional returns (including jotai hook calls useAtom and useAtomValue); when a user explicitly tells you a function handles null inputs, trust them and stop trying to "protect" it with unnecessary checks or workarounds.
- **Match response length to question complexity** - For simple, direct questions in Ask mode (especially those that can be answered in 1-2 sentences), provide equally brief answers. Save detailed explanations for complex topics or when explicitly requested.

### Strict Comment Rules

- **NEVER add comments that merely describe what code is doing**:
    - ❌ `mutex.Lock() // Lock the mutex`
    - ❌ `counter++ // Increment the counter`
    - ❌ `buffer.Write(data) // Write data to buffer`
    - ❌ `// Header component for app run list` (above AppRunListHeader)
    - ❌ `// Updated function to include onClick parameter`
    - ❌ `// Changed padding calculation`
    - ❌ `// Removed unnecessary div`
    - ❌ `// Using the model's width value here`
- **Only use comments for**:
    - Explaining WHY a particular approach was chosen
    - Documenting non-obvious edge cases or side effects
    - Warning about potential pitfalls in usage
    - Explaining complex algorithms that can't be simplified
- **When in doubt, leave it out**. No comment is better than a redundant comment.
- **Never add comments explaining code changes** - The code should speak for itself, and version control tracks changes. The one exception to this rule is if it is a very unobvious implementation. Something that someone would typically implement in a different (wrong) way. Then the comment helps us remember WHY we changed it to a less obvious implementation.

### Tool Use

Do NOT use write_to_file unless it is a new file or very short. Always prefer to use replace_in_file. Often your diffs fail when a file may be out of date in your cache vs the actual on-disk format. You should RE-READ the file and try to create diffs again if your diffs fail rather than fall back to write_to_file. If you feel like your ONLY option is to use write_to_file please ask first.

Also when adding content to the end of files prefer to use the new append_file tool rather than trying to create a diff (as your diffs are often not specific enough and end up inserting code in the middle of existing functions).

### Directory Awareness

- **ALWAYS verify the current working directory before executing commands**
- Either run "pwd" first to verify the directory, or do a "cd" to the correct absolute directory before running commands
- When running tests, do not "cd" to the pkg directory and then run the test. This screws up the cwd and you never recover. run the test from the project root instead.
