import {browser} from "$app/environment";
import {writable, type Writable} from "svelte/store";

export function persistedBoolean(key: string, defaultValue = false): Writable<boolean> {
    const store = writable(defaultValue);

    if (browser) {
        try {
            const raw = localStorage.getItem(key);
            if (raw !== null) {
                store.set(raw === "true");
            }
            store.subscribe((value) => {
                localStorage.setItem(key, value ? "true" : "false");
            });
        } catch {
            // If storage is unavailable, just keep in-memory state.
        }
    }

    return store;
}
