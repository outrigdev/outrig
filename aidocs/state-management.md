# State Management in Outrig

Outrig uses [Jotai](https://jotai.org/) for state management in the frontend. This document provides an overview of how state is managed in the application and best practices for working with Jotai.

## Overview

Jotai is an atomic state management library for React that focuses on simplicity and composability. In Outrig, we use Jotai to manage both global application state and component-local state.

The main advantages of Jotai in our application:

1. **Atomic updates**: Only components that use a specific atom will re-render when that atom changes
2. **Derived state**: Easily create computed values based on other atoms
3. **No boilerplate**: Minimal setup compared to other state management solutions
4. **TypeScript integration**: Strong typing support

## AppModel

The central state store for the application is defined in `frontend/appmodel.ts`. This singleton class contains atoms for:

- UI state (selected tab, dark mode)
- Status metrics (number of goroutines, log lines)
- Application data (app runs, logs, goroutines)

Example from AppModel:

```typescript
class AppModel {
    // UI state
    selectedTab: PrimitiveAtom<string> = atom("appruns");
    darkMode: PrimitiveAtom<boolean> = atom<boolean>(localStorage.getItem(ThemeLocalStorageKey) !== "light");

    // App runs data
    appRuns: PrimitiveAtom<AppRunInfo[]> = atom<AppRunInfo[]>([]);
    selectedAppRunId: PrimitiveAtom<string> = atom<string>("");
    appRunLogs: PrimitiveAtom<LogLine[]> = atom<LogLine[]>([]);

    // Methods to update state...
}
```

## Atom Types

### PrimitiveAtom vs. ReadonlyAtom

In Jotai, there are two main types of atoms:

1. **PrimitiveAtom**: Writable atoms that can be both read and updated
2. **Atom** (often used as ReadonlyAtom): Read-only atoms, typically derived from other atoms

When defining atoms that need to be updated, always use `PrimitiveAtom<Type>` rather than just `atom<Type>`:

```typescript
// Correct - explicitly typed as PrimitiveAtom
selectedTab: PrimitiveAtom<string> = atom("appruns");

// Incorrect - implicitly typed, may cause issues when updating
selectedTab = atom("appruns");
```

### Derived Atoms

Derived atoms (computed values) are created by passing a getter function to the atom constructor:

```typescript
filteredGoroutines: Atom<GoroutineData[]> = atom((get) => {
    const search = get(this.searchTerm);
    const showAll = get(this.showAll);
    const selectedStates = get(this.selectedStates);
    const goroutines = get(AppModel.appRunGoroutines);

    // Apply filters and return filtered data
    // ...
});
```

The getter function receives a `get` function that can be used to read other atoms. The derived atom will automatically update whenever any of the atoms it depends on change.

## Using Atoms in Components

### Reading Atom Values

To read an atom value in a component, use the `useAtomValue` hook:

```typescript
import { useAtomValue } from "jotai";
import { AppModel } from "../appmodel";

function MyComponent() {
    const selectedTab = useAtomValue(AppModel.selectedTab);
    // ...
}
```

### Reading and Writing Atom Values

To both read and write an atom value, use the `useAtom` hook:

```typescript
import { useAtom } from "jotai";
import { AppModel } from "../appmodel";

function MyComponent() {
    const [selectedTab, setSelectedTab] = useAtom(AppModel.selectedTab);

    const handleTabChange = (tab) => {
        setSelectedTab(tab);
    };
    // ...
}
```

### Component-Local State

For state that's specific to a component and doesn't need to be shared, you can create atoms within the component:

```typescript
function MyComponent() {
    const localState = useAtom(atom("initial value"));
    // ...
}
```

However, for better performance, it's recommended to define component-specific atoms outside the component using a model class:

```typescript
// my-component-model.ts
export class MyComponentModel {
    searchTerm: PrimitiveAtom<string> = atom("");
    // ...
}

// my-component.tsx
function MyComponent() {
    const model = useRef(new MyComponentModel()).current;
    const [search, setSearch] = useAtom(model.searchTerm);
    // ...
}
```

## Updating State with Side Effects

For state updates that involve side effects (like API calls), we use methods in the AppModel:

```typescript
// In AppModel
async loadAppRunGoroutines(appRunId: string) {
    if (!this.rpcClient) return;

    try {
        getDefaultStore().set(this.isLoadingGoroutines, true);
        const result = await RpcApi.GetAppRunGoroutinesCommand(this.rpcClient, { apprunid: appRunId });
        getDefaultStore().set(this.appRunGoroutines, result.goroutines);
        getDefaultStore().set(this.selectedAppRunId, appRunId);
    } catch (error) {
        console.error(`Failed to load goroutines for app run ${appRunId}:`, error);
    } finally {
        getDefaultStore().set(this.isLoadingGoroutines, false);
    }
}

// In a component
function MyComponent() {
    const handleRefresh = () => {
        if (selectedAppRunId) {
            AppModel.loadAppRunGoroutines(selectedAppRunId);
        }
    };
    // ...
}
```

## Global Store Access

In some cases, you may need to access the Jotai store outside of React components. For this, we use the `getDefaultStore` function and make it available globally:

```typescript
// In init.ts
import { getDefaultStore } from "jotai";

declare global {
    interface Window {
        jotaiStore: any;
    }
}
window.jotaiStore = getDefaultStore();
```

This allows us to access and update atoms from non-React code:

```typescript
// In a model class
toggleShowAll(): void {
    const store = window.jotaiStore;
    const showAll = store.get(this.showAll);

    if (!showAll) {
        // If enabling "show all", clear selected states
        store.set(this.selectedStates, new Set<string>());
    }

    store.set(this.showAll, !showAll);
}
```

## Best Practices

1. **Atom Granularity**: Create atoms at an appropriate level of granularity. Too fine-grained can lead to excessive re-renders, while too coarse-grained can cause unnecessary re-renders of components that don't need all the data.

2. **Derived State**: Use derived atoms for computed values rather than computing them in components.

3. **Model Classes**: For complex components, create a model class to encapsulate related atoms and logic.

4. **Explicit Typing**: Always explicitly type atoms as `PrimitiveAtom<Type>` when they need to be updated.

5. **Avoid Redundant State**: Don't duplicate state that can be derived from existing atoms.

6. **Performance Considerations**: For large lists or complex data structures, consider using memoization or virtualization to improve performance.

## Example: Filtering Pattern

A common pattern in Outrig is filtering data based on user input. Here's an example from the GoRoutines component:

```typescript
// In the model
class GoRoutinesModel {
    // State for filters
    searchTerm: PrimitiveAtom<string> = atom("");
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Derived state for available filter options
    availableStates: Atom<string[]> = atom((get) => {
        const goroutines = get(AppModel.appRunGoroutines);
        const statesSet = new Set<string>();

        goroutines.forEach((goroutine) => {
            statesSet.add(goroutine.state);
        });

        return Array.from(statesSet).sort();
    });

    // Derived state for filtered data
    filteredGoroutines: Atom<GoroutineData[]> = atom((get) => {
        const search = get(this.searchTerm);
        const showAll = get(this.showAll);
        const selectedStates = get(this.selectedStates);
        const goroutines = get(AppModel.appRunGoroutines);

        // Apply filters and return filtered data
        // ...
    });

    // Methods to toggle filters
    // ...
}

// In the component
function GoRoutines() {
    const model = useRef(new GoRoutinesModel()).current;
    const [search, setSearch] = useAtom(model.searchTerm);
    const [showAll, setShowAll] = useAtom(model.showAll);
    const filteredGoroutines = useAtomValue(model.filteredGoroutines);
    // ...
}
```

This pattern separates the filtering logic from the component, making it easier to test and maintain.
