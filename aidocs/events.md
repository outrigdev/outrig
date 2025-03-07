# Outrig Event System Documentation

This document describes the event PubSub system in Outrig, located at `@/rpc/rps.ts`.

## Overview

The event system enables components to publish and subscribe to events. Each event is associated with an `eventType` and optional `scopes`. Subscribers will receive all events matching their subscribed `eventType` and scopes.

## Event Types

Events in Outrig are typed using a discriminated union based on the `event` field. Each event type extends the `EventCommonFields` interface which may include:

```typescript
type EventCommonFields = {
    scopes?: string[]; // Optional scopes for more granular filtering
    sender?: string; // Optional sender information
    persist?: number; // Optional persistence configuration
};
```

Event types follow this pattern:

```typescript
type EventType =
    | (EventCommonFields & { event: "route:down"; data?: null })
    | (EventCommonFields & { event: "route:up"; data?: null })
    | (EventCommonFields & { event: "app:statusupdate"; data: StatusUpdateData });
// ... other event types
```

## Event Subscription

You subscribe to events using an event handler function along with an `eventType` and optional scopes:

```typescript
import { eventSubscribe } from "@/rpc/rps";

const eventHandler = (event: EventType) => {
    console.log("Event received:", event);
};

// Subscribe to all events of a specific type
const disposer = eventSubscribe({
    eventType: "app:statusupdate",
    handler: eventHandler,
});

// Subscribe to scoped events
const disposer = eventSubscribe({
    eventType: "user:update",
    handler: eventHandler,
    scope: "user:123",
});

// Later, unsubscribe when no longer needed
disposer();
```

The `eventSubscribe` function returns a disposer function that can be called to unsubscribe from the event. This is particularly useful in React components with `useEffect` to clean up subscriptions when the component unmounts:

```typescript
useEffect(() => {
    const disposer = eventSubscribe({
        eventType: "app:statusupdate",
        handler: handleStatusUpdate,
    });

    // Clean up subscription when component unmounts
    return disposer;
}, []);
```

## Scope Matching

Event scopes provide fine-grained control over subscriptions. Scope strings can use wildcard patterns:

- `*` matches exactly one segment
- `**` matches all remaining segments (must be at the end)

### Scope Matching Examples

| Pattern   | Matches                                             | Does not match     |
| --------- | --------------------------------------------------- | ------------------ |
| `user:*`  | `user:123`                                          | `user:123:profile` |
| `user:**` | matches all nested scopes like `user:login:success` |                    |

## Naming Conventions

When working with the event system, follow these naming conventions:

- Event names should use lowercase with colons as separators (e.g., `app:statusupdate`, `route:down`)
- Never use underscores in event names or scopes
- Scopes should follow the same pattern (e.g., `user:123`, `org:456:settings`)
