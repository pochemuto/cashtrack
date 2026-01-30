import {Auth} from "$lib/api";
import type {User as AuthUser} from "$lib/gen/api/v1/auth_pb";
import {Code, ConnectError} from "@connectrpc/connect";
import {writable} from "svelte/store";

export type User = AuthUser;

export const user = writable<User | undefined | null>(null);

export async function loadUser() {
    try {
        const response = await Auth.me({});
        if (response.user) {
            user.set(response.user);
            return response.user;
        }
        user.set(undefined);
    } catch (err) {
        if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
            user.set(undefined);
            return undefined;
        }
        user.set(undefined);
    }
    return undefined;
}

export async function logoutUser() {
    try {
        await Auth.logout({});
    } finally {
        user.set(undefined);
    }
}
