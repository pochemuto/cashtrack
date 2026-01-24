import {writable} from "svelte/store";

export type User = {
    id?: string;
    email?: string;
    name?: string;
};

export const user = writable<User | undefined>(undefined);
