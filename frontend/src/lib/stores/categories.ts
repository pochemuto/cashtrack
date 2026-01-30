import {Categories} from "$lib/api";
import type {Category} from "$lib/gen/api/v1/categories_pb";
import {Code, ConnectError} from "@connectrpc/connect";
import {get, writable} from "svelte/store";
import {user} from "../../user";

export const categories = writable<Category[]>([]);
export const categoriesLoading = writable(false);
export const categoriesError = writable("");

let loadedUserId: number | null = null;
let loadInFlight: Promise<boolean> | null = null;

function sortCategories(items: Category[]): Category[] {
    return [...items].sort((a, b) => a.name.localeCompare(b.name));
}

export function setCategories(items: Category[]) {
    categories.set(sortCategories(items));
}

export function addCategory(category: Category) {
    categories.update((items) => sortCategories([...items, category]));
}

export function updateCategory(updated: Category) {
    categories.update((items) =>
        sortCategories(items.map((category) => (category.id === updated.id ? updated : category)))
    );
}

export function removeCategory(categoryId: number) {
    categories.update((items) => items.filter((category) => category.id !== categoryId));
}

export async function loadCategories(force = false): Promise<boolean> {
    const currentUser = get(user);
    if (!currentUser || !currentUser.id) {
        loadedUserId = null;
        loadInFlight = null;
        categories.set([]);
        categoriesLoading.set(false);
        categoriesError.set("");
        return true;
    }
    if (!force && loadedUserId === currentUser.id) {
        return true;
    }
    if (!force && loadInFlight) {
        return loadInFlight;
    }

    categoriesLoading.set(true);
    categoriesError.set("");

    const loadPromise = (async () => {
        try {
            const response = await Categories.listCategories({});
            setCategories(response.categories ?? []);
            loadedUserId = currentUser.id;
            return true;
        } catch (err) {
            categories.set([]);
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                categoriesError.set("Failed to load categories.");
            } else {
                categoriesError.set("Failed to load categories.");
            }
            loadedUserId = null;
            return false;
        } finally {
            categoriesLoading.set(false);
            loadInFlight = null;
        }
    })();

    loadInFlight = loadPromise;
    return loadPromise;
}

user.subscribe((current) => {
    if (!current || !current.id) {
        loadedUserId = null;
        categories.set([]);
        categoriesLoading.set(false);
        categoriesError.set("");
    }
});
