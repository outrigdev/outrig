# Backend RPC Events System Guide

This guide explains how to work with the Outrig backend event system, which is built on top of the RPC system. It's designed to help AI agents understand and implement event-based functionality in the Outrig backend.

## Overview

The Outrig event system allows different components to communicate asynchronously through events. It consists of:

1. **Event Broker**: Central component that manages event subscriptions and dispatches events
2. **Event Publishers**: Components that publish events
3. **Event Subscribers**: Components that subscribe to and handle events

Events are identified by their event type (e.g., `route:down`) and can include additional data.

## Creating an Event Handler

To create a component that handles events, follow these steps:

### 1. Create an RPC Client for Your Component

```go
// Define a constant for your route ID
const YourComponentRouteId = "yourcomponent"

// Create an RPC client for your component
var yourComponentRpcClient *rpc.RpcClient

// Initialize your component
func Initialize() {
    // Create a new RPC client
    yourComponentRpcClient = rpc.MakeRpcClient(nil, nil, nil, YourComponentRouteId)
    
    // Register the client with the router
    rpc.DefaultRouter.RegisterRoute(YourComponentRouteId, yourComponentRpcClient, true)
    
    // Set up event handling (see next steps)
}
```

### 2. Subscribe to Events

Subscribe to the events you want to handle:

```go
// Subscribe to an event
rpc.Broker.Subscribe(YourComponentRouteId, rpctypes.SubscriptionRequest{
    Event:     rpctypes.Event_YourEventType, // e.g., rpctypes.Event_RouteDown
    AllScopes: true, // Set to true to receive events for all scopes
})
```

### 3. Register Event Handlers

Register handlers for the events you've subscribed to:

```go
// Register an event handler
yourComponentRpcClient.EventListener.On(rpctypes.Event_YourEventType, func(event *rpctypes.EventType) {
    if event == nil {
        return
    }
    
    // Extract data from the event
    // For example, for route:down events, the route ID is in event.Sender
    routeId := event.Sender
    
    // Handle the event
    // ...
})
```

## Publishing Events

To publish events from your component:

```go
// Create an event
event := rpctypes.EventType{
    Event:  rpctypes.Event_YourEventType,
    Sender: YourComponentRouteId,
    Data:   yourEventData, // Optional data to include with the event
}

// Publish the event
rpc.Broker.Publish(event)
```

## Event Persistence

Events can be persisted for a specified duration, allowing new subscribers to receive past events:

```go
// Create an event with persistence
event := rpctypes.EventType{
    Event:   rpctypes.Event_YourEventType,
    Sender:  YourComponentRouteId,
    Persist: 10, // Number of events to persist
    Data:    yourEventData,
}
```

## Common Events

Some common events in the Outrig system:

- `route:down`: Triggered when an RPC route is unregistered (e.g., when a connection closes)
- `route:up`: Triggered when an RPC route is registered
- `app:statusupdate`: Triggered when an app's status changes

## Example: Handling Route Down Events

Here's a complete example of handling route down events:

```go
package yourcomponent

import (
    "log"
    "sync"

    "github.com/outrigdev/outrig/pkg/rpc"
    "github.com/outrigdev/outrig/pkg/rpctypes"
)

// Constants
const (
    YourComponentRouteId = "yourcomponent"
)

// Global state
var (
    yourMutex sync.Mutex
    yourData  = make(map[string]YourDataType)
)

// RPC client for your component
var yourComponentRpcClient *rpc.RpcClient

// Initialize sets up your component
func Initialize() {
    // Create a new RPC client
    yourComponentRpcClient = rpc.MakeRpcClient(nil, nil, nil, YourComponentRouteId)
    
    // Register the client with the router
    rpc.DefaultRouter.RegisterRoute(YourComponentRouteId, yourComponentRpcClient, true)
    
    // Subscribe to route down events
    rpc.Broker.Subscribe(YourComponentRouteId, rpctypes.SubscriptionRequest{
        Event:     rpctypes.Event_RouteDown,
        AllScopes: true,
    })
    
    // Register an event handler for route down events
    yourComponentRpcClient.EventListener.On(rpctypes.Event_RouteDown, func(event *rpctypes.EventType) {
        if event == nil {
            return
        }
        
        // Extract the route ID from the event
        routeId := event.Sender
        if routeId != "" {
            log.Printf("[yourcomponent] Route down event for %s", routeId)
            
            // Clean up any data associated with this route
            yourMutex.Lock()
            delete(yourData, routeId)
            yourMutex.Unlock()
        }
    })
    
    log.Printf("[yourcomponent] Initialized and subscribed to route down events")
}
```

## Tips for AI Agents

1. **Route IDs**: Each component should have a unique route ID. Use a constant for this.
2. **Thread Safety**: Always use mutexes when accessing shared data structures.
3. **Null Checks**: Always check if event or event data is nil before accessing it.
4. **Logging**: Use log.Printf with a component prefix for easier debugging.
5. **Initialization**: Initialize your component in the server's main function.

## Debugging Events

If you're having trouble with events:

1. Check that your component is properly registered with the router
2. Verify that you've subscribed to the correct event type
3. Ensure your event handler is registered correctly
4. Add logging to confirm events are being received
5. Check for any errors in the event subscription or publishing process

By following this guide, you should be able to effectively work with the Outrig backend event system.
