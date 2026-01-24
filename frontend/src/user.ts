import {writable} from "svelte/store";
import {resolveApiUrl} from "$lib/url";

export type User = {
    id: number;
    username: string;
};

export const user = writable<User | undefined | null>(null);

export async function loadUser() {
    try {
        const response = await fetch(resolveApiUrl("auth/me"), {
            credentials: "include",
        });
        if (response.ok) {
            const data = (await response.json()) as User;
            user.set(data);
            return data;
        }
        if (response.status === 401) {
            user.set(undefined);
        }
    } catch {
        user.set(undefined);
    }
    return undefined;
}

export async function logoutUser() {
    try {
        await fetch(resolveApiUrl("auth/logout"), {
            method: "POST",
            credentials: "include",
        });
    } finally {
        user.set(undefined);
    }
}
