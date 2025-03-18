# Keyboard Event Handling in Outrig

This document describes how to handle keyboard events in the Outrig application using the utilities provided in `frontend/util/keyutil.ts`.

## Key Concepts

Outrig provides a consistent way to handle keyboard events across different platforms (macOS, Windows, Linux) through the following key utilities:

- `OutrigKeyboardEvent`: A normalized keyboard event that abstracts away browser differences
- `keydownWrapper`: A utility function that simplifies keyboard event handling
- `checkKeyPressed`: A function to check if a specific key or key combination was pressed
- Key descriptions: A string format for describing key combinations (e.g., "Cmd:ArrowDown")

## Platform Compatibility

Outrig's keyboard handling automatically maps keys across platforms:

- `Cmd` maps to the Command key on macOS and Alt key on Windows/Linux
- `Option` maps to the Option/Alt key on macOS and Meta key on Windows/Linux
- `Meta` maps to the Command key on macOS and Windows key on Windows/Linux
- `Alt` maps to the Option/Alt key on macOS and Alt key on Windows/Linux

## Using keydownWrapper

The `keydownWrapper` function is the recommended way to handle keyboard events. It:

1. Converts native or React keyboard events to `OutrigKeyboardEvent`
2. Handles `preventDefault()` and `stopPropagation()` automatically when your handler returns `true`
3. Provides a cleaner syntax for keyboard event handling

### Example Usage

```typescript
import { keydownWrapper, checkKeyPressed } from "@/util/keyutil";

// In a React component
const handleKeyDown = useCallback(
    keydownWrapper((keyEvent: OutrigKeyboardEvent) => {
        if (checkKeyPressed(keyEvent, "Cmd:ArrowDown")) {
            // Handle Cmd+ArrowDown
            doSomething();
            return true; // Returning true prevents default and stops propagation
        }

        if (checkKeyPressed(keyEvent, "PageUp")) {
            // Handle PageUp
            doSomethingElse();
            return true;
        }

        return false; // Event not handled, continue normal processing
    }),
    [dependencies]
);

// Attach to a component
<div onKeyDown={handleKeyDown}>...</div>
```

## Key Descriptions

Key descriptions are strings that describe key combinations. They consist of modifier keys and a main key, separated by colons:

```
[Modifier1]:[Modifier2]:...[Key]
```

### Supported Modifiers

- `Cmd`: Command key on macOS, Alt key on Windows/Linux
- `Shift`: Shift key
- `Ctrl`: Control key
- `Option`: Option/Alt key on macOS, Meta key on Windows/Linux
- `Alt`: Alt key on Windows/Linux, Option key on macOS
- `Meta`: Command key on macOS, Windows key on Windows/Linux

### Key Types

There are two ways to specify the key in a key description:

1. **Key Value (`event.key`)**: By default, key descriptions use the `key` property from the keyboard event. These are the values like "a", "A", "ArrowDown", "Enter", etc.

2. **Key Code (`event.code`)**: You can also use the `code` property by using the `c{}` syntax. For example, `"c{KeyA}"` refers to the A key regardless of the keyboard layout.

### Examples

- `"Cmd:ArrowDown"`: Command+Down Arrow on macOS, Alt+Down Arrow on Windows/Linux
- `"Shift:A"`: Shift+A (uppercase A)
- `"Ctrl:c"`: Control+C
- `"Cmd:Shift:z"`: Command+Shift+Z on macOS, Alt+Shift+Z on Windows/Linux
- `"c{KeyA}"`: The A key using the key code
- `"Shift:c{Digit1}"`: Shift+1 using the key code

## Using checkKeyPressed

The `checkKeyPressed` function checks if a keyboard event matches a key description:

```typescript
if (checkKeyPressed(keyEvent, "Cmd:ArrowDown")) {
    // This code runs when Cmd+ArrowDown is pressed
}

// Using key code
if (checkKeyPressed(keyEvent, "c{KeyS}")) {
    // This code runs when the S key is pressed (using key code)
}
```

## Global Keyboard Shortcuts

For global keyboard shortcuts that work throughout the application, use the `keymodel.ts` module:

1. Add your shortcut to the `registerGlobalKeys` function in `frontend/keymodel.ts`
2. The shortcut will be available throughout the application

## Component-Specific Shortcuts

For shortcuts that only apply to a specific component:

1. Use `keydownWrapper` and `checkKeyPressed` as shown above
2. Attach the handler to the appropriate element in your component

## Key vs. Code

- `event.key`: Represents the character that would be produced by the key press. It's affected by keyboard layout, language, and modifier keys.
- `event.code`: Represents the physical key on the keyboard, regardless of the character it would produce. It's not affected by keyboard layout or language.

When to use which:

- Use `key` (default) when you care about the character being typed (e.g., "a", "A", "ArrowDown")
- Use `code` with `c{}` syntax when you care about the physical key position (e.g., "c{KeyA}" for the A key regardless of layout)

## Best Practices

1. Use `keydownWrapper` for all keyboard event handling
2. Return `true` from your handler when you've handled the event
3. Use descriptive key combinations that make sense across platforms
4. For text inputs, consider what keys should be handled vs. passed through
5. Keep keyboard shortcuts consistent throughout the application
6. Document keyboard shortcuts in the UI where appropriate
