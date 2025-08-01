# Outrig

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.png" width="300">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-light.png" width="300">
    <img src="assets/outriglogo.png" alt="Outrig Logo" width="300">
  </picture>
</p>

Outrig is an open-source observability tool for Go development. It provides real-time log search, goroutine monitoring, variable tracking, and runtime metrics to help you debug and understand your applications.

Outrig runs 100% locally. No data ever leaves your machine.

It is currently available for macOS and Linux.

- **Homepage**: https://outrig.run
- **Discord** (new): https://discord.gg/u9gByfvZm9

<p align="center">
  <img src="assets/outrig-loop.gif" alt="Outrig in action" width="800">
</p>

[![Go Reference](https://pkg.go.dev/badge/github.com/outrigdev/outrig.svg)](https://pkg.go.dev/github.com/outrigdev/outrig)
[![Go Report Card](https://goreportcard.com/badge/github.com/outrigdev/outrig)](https://goreportcard.com/report/github.com/outrigdev/outrig)
![Go Version](https://img.shields.io/github/go-mod/go-version/outrigdev/outrig)

## Features

- **Real-time Log Viewing**: Stream and search logs from your Go application in real-time
- **Goroutine Monitoring**: Track and inspect goroutines, including custom naming
- **Variable Watching**: Monitor variables and counters in your application
- **Runtime Hooks**: Execute hooks in your running application (coming soon)
- **Minimal Integration**: Integrate into your Go application in seconds
- **Docker Integration**: See [Using Docker in the Outrig Docs](https://outrig.run/docs/using-docker)

## How It Works

Outrig consists of two main components that work together:

1. **SDK Client**: A lightweight Go library that you import into your application. It collects logs, goroutine information, and other runtime data from your application and sends it to the Outrig Monitor. [API Docs](https://pkg.go.dev/github.com/outrigdev/outrig)

2. **Outrig Monitor**: A standalone server that receives data from your application, processes it, and provides a web-based dashboard for real-time monitoring and debugging.

## Installation

For macOS:

```bash
brew install --cask outrigdev/outrig/outrig
```

This installs a system tray application. After installation, you'll need to launch the Outrig application from your Applications folder or Spotlight to start the server. Once you run the application the first time, the `outrig` CLI tool will be available in your path.

For Linux:

```bash
# Quick installation script (installs to ~/.local/bin)
curl -sf https://outrig.run/install.sh | sh
```

Alternatively, you can download .dmg, .deb, .rpm, or .tar.gz packages directly from our [GitHub releases page](https://github.com/outrigdev/outrig/releases).

For Nix users:

```bash
# Install using the flake
nix profile install github:outrigdev/outrig
```

For developers interested in building from source, see [BUILD.md](docs/BUILD.md). If you've already cloned the repository, you can build and install with:

```bash
# Build from source and install to ~/.local/bin
task install
```

## Usage

The simplest way to try Outrig is by running your `main.go` using the Outrig CLI:

```bash
outrig run main.go
```

The `outrig run` command takes all the same arguments as `go run` (as it is implemented as a thin wrapper around `go run`). Under the hood `outrig run` instruments your code to work with the Outrig monitor, giving you log searching, a full goroutine viewer, and runtime stats out of the box.

### Integration using the SDK

You can also integrate Outrig by adding a single import to your Go application's main file:

```go
// Add this import to your main Go file:
import _ "github.com/outrigdev/outrig/autoinit"

// That's it! Your app will appear in Outrig when run
```

### Running the Outrig Monitor

**macOS**

The Outrig Monitor is managed through the system tray application. After installation, launch the Outrig app from your Applications folder or Spotlight. The monitor will start automatically and you'll see the Outrig icon in your system tray.

**Linux**

To start the Outrig Monitor, run the following command in your terminal:

```bash
outrig server
```

To stop the monitor, use Ctrl+C in the terminal where the monitor is running. Note that future versions will include systemd support to run the monitor as a background service.

To verify the monitor is running correctly, navigate to http://localhost:5005 and you should see the Outrig dashboard.

## Key Features

### Logs

Outrig captures and displays logs from your Go application in real-time out of the box by capturing stdout/stderr.

```go
// Logs are automatically captured from stdout and stderr
fmt.Println("This will be captured by Outrig")
log.Printf("Standard Go logging is captured too")
```

Features:

- Real-time log streaming
- Instant type-ahead progressive searching
- Advanced search and filtering capabilities (exact match, fuzzy search, regexp, tags, ANDs, and ORs)
- Follow mode to automatically track latest logs

### Watches

Easily monitor variables in your application. Outrig can display structures (JSON or %#v output) and numeric values (easy graphing and historical data viewing coming soon). Values are collected automatically every second (except for push-based watches).

```go
// Basic watch using a function
outrig.NewWatch("counter").PollFunc(func() int {
    return myCounter
})

// Watch with mutex protection
var mu sync.Mutex
var counter int
outrig.NewWatch("sync-counter").PollSync(&mu, &counter)

// Watch an atomic value
var atomicCounter atomic.Int64
outrig.NewWatch("atomic-counter").PollAtomic(&atomicCounter)

// Push values directly from your code
pusher := outrig.NewWatch("requests").ForPush()
pusher.Push(42)
// Later...
pusher.Push(43)

// Chained configuration with tags and formatting
outrig.NewWatch("api-response").
    WithTags("api", "performance").
    AsJSON().
    PollFunc(func() interface{} {
        return app.GetAPIStats()
    })

// Use as a counter (incremental values)
outrig.NewWatch("request-count").
    WithTags("performance").
    AsCounter().
    PollFunc(getRequestCount)
```

### Goroutine Monitoring

Outrig captures your goroutine stack traces every second for easy search/viewing. You can optionally name and tag your goroutines for easier inspecting.

When using the `outrig run` CLI, you can name any goroutine in your own module by using a special comment above the `go` keyword. This will override the system generated name for the goroutine in the outrig viewer.

```go
//outrig name="worker-thread"
go func() {
	...
}()
```

Alternatively you can use the SDK to give your goroutines names and tags:

```go
outrig.Go("worker-pool-1").WithTags("worker").Run(func() {
    // Goroutine code...
})
```

### Runtime Stats

Outrig gathers runtime stats every second. Including:

- Memory usage breakdown with visual charts
- CPU usage monitoring
- Goroutine count tracking
- Heap memory allocation statistics
- Garbage collection cycle monitoring
- Process information display (PID, uptime, etc.)
- Go runtime version and environment details

## Architecture

The Outrig codebase is organized into three main components:

1. **Client SDK** (`outrig.go` and `pkg/`): A lightweight Go library that integrates with your application. It collects logs, goroutine information, and other runtime data and sends it to the Outrig Monitor.

2. **Monitor** (`server/`): A standalone Go server that receives data from your application, processes it, and exposes it via an RPC API. The monitor efficiently stores and indexes the data for quick searching and retrieval. It has a separate go.mod file so its dependencies don't pollute the SDK.

3. **Frontend** (`frontend/`): A React TypeScript application that communicates with the monitor via WebSocket using RPC calls. It provides a user-friendly interface for monitoring and debugging your application in real-time. It is built and embedded into the Outrig Monitor.

### Data Flow

1. Your Go application imports the Outrig SDK and initializes it with the autoinit package or an explicit call to `outrig.Init()`
2. The SDK collects logs, goroutine information, and other runtime data
3. This data is sent to the Outrig Monitor via a local domain socket
4. The monitor server processes and stores the data
5. Go to http://localhost:5005 to view and interact with your data

### Performance

- **Minimal overhead by design** — When disconnected, the SDK enters standby mode, suspends collection, and performs only a brief periodic check for reconnection.
- **Disable in Production** — A build flag (+no_outrig) can completely disable SDK at compile time

### Security

- **No open ports** — Your program doesn't expose extra HTTP servers or ports. It attempts to make a domain socket connection to the Outrig Monitor. If the domain socket is not found or is not active, the SDK will remain in standby mode
- **Secure by default** — All connections stay on localhost (unless you explicitly configure it otherwise); no application data leaves your machine

### Telemetry

To help understand how many people are using Outrig, help prioritize new features, and find/fix bugs we collect _minimal_ anonymous telemetry from the Outrig Monitor. It can be disabled on the CLI by running `outrig server --no-telemetry`. Note that the SDK does not send _any_ telemetry.

## Development

For information on building from source, setting up a development environment, and contributing to Outrig, see [BUILD.md](docs/BUILD.md).

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.
