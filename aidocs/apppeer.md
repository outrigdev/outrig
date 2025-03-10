# AppRunPeer

The AppRunPeer represents a connection to a running Go application in the Outrig system. This document describes its functionality and usage.

## Overview

Each AppRunPeer instance tracks:
- A unique application run ID
- Application information (name, PID, start time, etc.)
- Logs from the application
- Goroutine information
- Application status

## Application Status Tracking

The AppRunPeer tracks the status of the monitored application:

- **running**: The application is currently running and connected.
- **done**: The application has gracefully shut down using `outrig.AppDone()`.
- **disconnected**: The connection was closed without receiving an AppDone packet.

## AppDone Functionality

The `outrig.AppDone()` function allows applications to signal when they're shutting down gracefully. This helps distinguish between applications that have completed their work versus those that crashed or were forcibly terminated.

### Usage

In your Go application, after initializing Outrig, defer the AppDone call in your main function:

```go
func main() {
    // Initialize Outrig
    outrig.Init(nil)
    
    // Defer AppDone to signal when the application exits
    defer outrig.AppDone()
    
    // Rest of your application code
    // ...
}
```

### Implementation Details

When `outrig.AppDone()` is called:

1. It sends an "appdone" packet to the Outrig server
2. The server updates the AppRunPeer's status to "done"
3. This status can be used by the frontend to display the application's current state

If a connection is closed without receiving an AppDone packet, the server marks the AppRunPeer's status as "disconnected" instead.

## Connection Handling

The server automatically tracks connection status:

- When a new connection is established, the status is set to "running"
- When a connection is closed, if no AppDone packet was received, the status is set to "disconnected"
- If an AppDone packet is received, the status remains "done" even after the connection closes

This status tracking provides valuable information about how applications terminated, helping developers distinguish between normal shutdowns and potential crashes.
