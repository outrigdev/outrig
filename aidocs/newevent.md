# Defining a New Event

To define a new event within the RPC system, follow these steps:

## 1. Update `rpctypes.go`

Define a constant for the new event, using a descriptive name. For example:

```go
const Event_FeatureToggle = "featuretoggle"
```

Note that event string names should be all lowercase, no underscores. They may include a colon for namespacing if we're defining a set of events that are linked together in some way.

## 2. Define the Event Data Structure

If your event includes additional data, define a struct within `rpctypes.go`, including appropriate JSON tags:

```go
type FeatureToggleData struct {
	FeatureName string `json:"featurename"`
	Enabled     bool   `json:"enabled"`
}
```

## 2. Update `EventToTypeMap`

Map your new event constant to the corresponding data struct. The mapped struct specifies the data type expected in the `Data` field of `EventType`. If your event does not require additional data, set it to `nil`:

```go
EventToTypeMap = map[string]reflect.Type{
	Command_FeatureToggle: reflect.TypeOf(FeatureToggleData{}),
	// other events...
}
```

## Event Structure

All events share a common structure:

```go
type EventType struct {
	Event   string   `json:"event"`
	Scopes  []string `json:"scopes,omitempty"`
	Sender  string   `json:"sender,omitempty"`
	Persist int      `json:"persist,omitempty"`
	Data    any      `json:"data,omitempty"`
}
```

The `Data` field's concrete type is determined by the corresponding entry in `EventToTypeMap`.

## 3. Generate Code

After defining your event, regenerate TypeScript definitions (in rpctypes.d.ts). In TypeScript, EventType is a discriminated type union (using "event" as the discrimator), and then using EventToTypeMap for the "data" type.

```bash
task generate
```

Your new event is now ready to be integrated within the RPC system.
