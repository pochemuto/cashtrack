import {get, writable} from "svelte/store";
import {resolveApiUrl} from "$lib/url";
import {user} from "../../user";

export type CategoryItem = {
    id: number;
    name: string;
    color: string | null;
    created_at: string;
};

export const categories = writable<CategoryItem[]>([]);
export const categoriesLoading = writable(false);
export const categoriesError = writable("");

let loadedUserId: number | null = null;
let loadInFlight: Promise<boolean> | null = null;

function sortCategories(items: CategoryItem[]): CategoryItem[] {
    return [...items].sort((a, b) => a.name.localeCompare(b.name));
}

export function setCategories(items: CategoryItem[]) {
    categories.set(sortCategories(items));
}

export function addCategory(category: CategoryItem) {
    categories.update((items) => sortCategories([...items, category]));
}

export function updateCategory(updated: CategoryItem) {
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
            const response = await fetch(resolveApiUrl("api/categories"), {credentials: "include"});
            if (!response.ok) {
                categories.set([]);
                categoriesError.set("Failed to load categories.");
                loadedUserId = null;
                return false;
            }
            const payload = (await response.json()) as CategoryItem[];
            setCategories(payload);
            loadedUserId = currentUser.id;
            return true;
        } catch {
            categories.set([]);
            categoriesError.set("Failed to load categories.");
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
