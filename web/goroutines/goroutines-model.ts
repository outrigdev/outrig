import { Atom, atom, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";

class GoRoutinesModel {
    searchTerm: PrimitiveAtom<string> = atom("");

    // State filters
    showAll: PrimitiveAtom<boolean> = atom(true);
    selectedStates: PrimitiveAtom<Set<string>> = atom(new Set<string>());

    // Derived atom for all available states
    availableStates: Atom<string[]> = atom((get) => {
        const goroutines = get(AppModel.appRunGoRoutines);
        const statesSet = new Set<string>();

        goroutines.forEach((goroutine) => {
            statesSet.add(goroutine.state);
        });

        return Array.from(statesSet).sort();
    });

    // Filtered goroutines based on search term and state filters
    filteredGoroutines: Atom<GoroutineData[]> = atom((get) => {
        const search = get(this.searchTerm);
        const showAll = get(this.showAll);
        const selectedStates = get(this.selectedStates);
        const goroutines = get(AppModel.appRunGoRoutines);

        // First sort by goroutine ID
        const sortedGoroutines = [...goroutines].sort((a, b) => a.goid - b.goid);

        // Apply state filters if not showing all
        let stateFiltered = sortedGoroutines;
        if (!showAll && selectedStates.size > 0) {
            stateFiltered = sortedGoroutines.filter((goroutine) => selectedStates.has(goroutine.state));
        }

        // Apply search filter if there's a search term
        if (!search) {
            return stateFiltered;
        }

        return stateFiltered.filter(
            (goroutine) =>
                goroutine.stacktrace.toLowerCase().includes(search.toLowerCase()) ||
                goroutine.state.toLowerCase().includes(search.toLowerCase()) ||
                goroutine.goid.toString().includes(search)
        );
    });

    // Toggle a state filter
    toggleStateFilter(state: string): void {
        const store = window.jotaiStore;
        const selectedStates = store.get(this.selectedStates);
        const newSelectedStates = new Set(selectedStates);

        if (selectedStates.has(state)) {
            // Remove the state if it's already selected
            newSelectedStates.delete(state);

            // If no states are selected anymore, enable "show all"
            if (newSelectedStates.size === 0) {
                store.set(this.showAll, true);
            }
        } else {
            // Add the state and disable "show all"
            newSelectedStates.add(state);
            store.set(this.showAll, false);
        }

        store.set(this.selectedStates, newSelectedStates);
    }

    // Toggle "show all" filter
    toggleShowAll(): void {
        const store = window.jotaiStore;
        const showAll = store.get(this.showAll);

        if (!showAll) {
            // If enabling "show all", clear selected states
            store.set(this.selectedStates, new Set<string>());
        }

        store.set(this.showAll, !showAll);
    }
}

export { GoRoutinesModel };
