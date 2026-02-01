<script lang="ts">
    import {createEventDispatcher, onMount} from "svelte";
    import type {Category} from "$lib/gen/api/v1/categories_pb";
    import CategoryColorPicker from "$lib/components/CategoryColorPicker.svelte";
    import CategoryBadge from "$lib/components/CategoryBadge.svelte";

    export let open = false;
    export let title = "Категория";
    export let name = "";
    export let color: string | null = null;
    export let categories: Category[] = [];
    export let parentId: number | null = null;
    export let isGroup = false;
    export let selfId: number | null = null;
    export let showParent = true;
    export let showGroupToggle = true;
    export let confirmLabel = "Сохранить";
    export let saving = false;

    const dispatch = createEventDispatcher();
    let nameInput: {focus: () => void} | null = null;
    let parentIdValue = "";
    let parentOptions: Category[] = [];

    function handleCancel() {
        dispatch("cancel");
    }

    function handleSave() {
        dispatch("save");
    }

    function handleParentChange(event: Event) {
        const value = (event.currentTarget as HTMLSelectElement).value;
        parentId = value ? Number(value) : null;
    }

    function handleKeydown(event: KeyboardEvent) {
        if (!open) {
            return;
        }
        if (event.key === "Escape") {
            event.preventDefault();
            handleCancel();
        }
    }

    $: if (open) {
        queueMicrotask(() => nameInput?.focus());
    }

    $: parentOptions = categories.filter((category) => category.id !== selfId);

    $: parentIdValue = parentId ? String(parentId) : "";

    $: if (parentId !== null && !parentOptions.some((category) => category.id === parentId)) {
        parentId = null;
    }

    onMount(() => {
        window.addEventListener("keydown", handleKeydown);
        return () => window.removeEventListener("keydown", handleKeydown);
    });
</script>

{#if open}
    <div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="category-editor-title">
        <div class="modal-box">
            <h3 id="category-editor-title" class="text-lg font-semibold">{title}</h3>
            <form class="mt-4 space-y-4" on:submit|preventDefault={handleSave}>
                <div class="form-control w-full">
                    <CategoryBadge
                        bind:this={nameInput}
                        bind:name={name}
                        color={color ?? ""}
                        editable={true}
                        placeholder="Название категории"
                        className="badge-lg"
                    />
                </div>
                <div class="form-control w-full">
                    <div class="label">
                        <button
                            class="btn btn-ghost btn-xs"
                            type="button"
                            on:click={() => (color = null)}
                            disabled={!color}
                        >
                            Без цвета
                        </button>
                    </div>
                    <CategoryColorPicker bind:hex={color} label="Цвет" inline={true} nullable={false} />
                </div>
                {#if showParent}
                    <div class="form-control w-full">
                        <label class="label">
                            <span class="label-text">Родительская категория</span>
                        </label>
                        <select
                            class="select select-bordered"
                            bind:value={parentIdValue}
                            on:change={handleParentChange}
                            disabled={!parentOptions.length}
                        >
                            <option value="">Без родителя</option>
                            {#each parentOptions as category}
                                <option value={String(category.id)}>
                                    {category.name}{category.isGroup ? " (группа)" : ""}
                                </option>
                            {/each}
                        </select>
                    </div>
                {/if}
                {#if showGroupToggle}
                    <label class="label cursor-pointer gap-2">
                        <input class="checkbox checkbox-sm" type="checkbox" bind:checked={isGroup} />
                        <span class="label-text">Групповая категория</span>
                    </label>
                {/if}
                <div class="modal-action">
                    <button
                        class="btn btn-primary"
                        type="submit"
                        disabled={saving || !name.trim()}
                    >
                        {confirmLabel}
                    </button>
                    <button class="btn btn-ghost" type="button" on:click={handleCancel}>
                        Отмена
                    </button>
                </div>
            </form>
        </div>
        <button class="modal-backdrop" type="button" on:click={handleCancel} aria-label="close"></button>
    </div>
{/if}
