# Building and Development Guide for Outrig

This document describes how to set up a local development environment for Outrig and build the project.

## Prerequisites

Before you begin, ensure you have the following installed:

1. **Go** (version 1.23.4 or later)

    ```bash
    # Check your Go version
    go version
    ```

2. **Node.js** (version 22 or later) and npm

    ```bash
    # Check your Node.js and npm versions
    node --version
    npm --version
    ```

3. **Task** (task runner)

    ```bash
    # macOS
    brew install go-task

    # Linux
    sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
    ```

## Project Structure

Outrig is organized into three main components:

1. **Client SDK** (`outrig.go` and `pkg/`): Go library

    - Main library at project root
    - Additional SDK packages in `pkg/`
    - Data structures in `ds.go`
    - Main coordination in `controller.go`

2. **Server** (`server/`): Go server

    - Entry point at `server/main-server.go`
    - Server-specific packages in `server/pkg/`

3. **Frontend** (`frontend/`): React TypeScript application
    - Uses Jotai for state management
    - Tailwind CSS v4 for styling
    - Vite for development and building

## Setting Up the Development Environment

1. Clone the repository:

    ```bash
    git clone https://github.com/outrigdev/outrig.git
    cd outrig
    ```

2. Install dependencies:
    ```bash
    npm install
    ```

## Development Workflow

### Running the Development Server

To run the development server:

```bash
task dev
```

This command:

- Starts the Go server in development mode
- Automatically runs the Vite development server for the frontend

You can then access the application at http://localhost:5173

Note: There's no need to run the frontend and backend separately - the `task dev` command handles everything you need for local development.

**Important**: The development version of Outrig sends data to the production version of Outrig for debugging purposes. This helps identify and fix issues in the development build.

### Generating Test Data

To generate test data for development:

```bash
task testrun     # Generate fake data
# or
task testsmall   # Generate minimal test data
```

## Code Generation

Outrig uses code generation for both Go and TypeScript code. After modifying Go types in `pkg/rpctypes/rpctypes.go`, run:

```bash
task generate
```

This will update:

- Go code for RPC client implementation in `pkg/rpcclient/rpcclient.go`
- TypeScript type definitions in `frontend/types/rpctypes.d.ts`
- RPC client API in `frontend/rpc/rpcclientapi.ts`

**Important**: Do not manually edit generated files. Instead, modify the source Go types and run the generate task.

## Building for Production

To build the entire project for production:

```bash
task build
```

This will:

1. Clean build artifacts
2. Build the frontend
3. Build the Go server with the embedded frontend

The production binary will be available at `bin/outrig`.

### Running the Production Build

To run the production build:

```bash
task prod
```

Or directly:

```bash
./bin/outrig server
```

## Production Releases

Outrig uses GoReleaser for production builds and releases. This is primarily handled by the maintainers, but if you're interested in the release process, see the `.goreleaser.yaml` configuration file.

## Troubleshooting

If you encounter issues:

1. Make sure all prerequisites are installed correctly
2. Try cleaning the build artifacts: `task clean`
3. Check for any error messages in the terminal
4. Ensure you're using the correct versions of Go and Node.js
