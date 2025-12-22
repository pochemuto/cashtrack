<script lang="ts">
    import {Todo} from "$lib/api";
    import type {ListItem} from "$lib/gen/api/v1/todo_pb"
    import {onMount} from "svelte";

    type Item = {
        id: number,
        title: string
        removing: boolean
    }

    function fromServer(item: ListItem): Item {
        return {
            id: item.id,
            title: item.title,
            removing: false
        }
    }

    function applyServer(serverItems: ListItem[]) {
        const prevRemoving = new Map<number, boolean>((items ?? []).map(i => [i.id, i.removing]));
        items = serverItems.map((s) => ({
            id: s.id,
            title: s.title,
            removing: prevRemoving.get(s.id) ?? false,
        }));
    }

    let items = $state<Item[] | null>(null);
    let loading = $state(false);
    let newItem = $state("");
    let opLoading = $state(false);
    let input: HTMLInputElement;

    onMount(load);

    async function load() {
        loading = true;
        try {
            const response = await Todo.list({});
            applyServer(response.items);
        } finally {
            loading = false;
        }
    }

    async function remove(item: Item) {
        item.removing = true;
        const response = await Todo.remove({id: item.id});
        applyServer(response.items);
    }

    async function add() {
        if (!newItem) {
            return;
        }
        opLoading = true;
        try {
            const response = await Todo.add({items: [{title: newItem}]});
            applyServer(response.items);
            newItem = "";
            input.focus();
        } finally {
            opLoading = false;
        }
    }

    async function random() {
        opLoading = true;
        try {
            const response = await Todo.addRandom({});
            applyServer(response.items);
            input.focus();
        } finally {
            opLoading = false;
        }
    }
</script>

<svelte:head>
    <title>Todo</title>
</svelte:head>


<section class="mx-auto w-full max-w-2xl">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-4">

            {#if loading}
                <p>Loading...</p>
            {/if}

            {#if items}
                <ul>
                    {#each items as item}
                        <li class="item" class:removing={item.removing}>
                            <button class="btn btn-xs btn-secondary btn-ghost btn-circle"
                                    class:btn-disabled={item.removing}
                                    onclick={() => remove(item)}>
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
                                </svg>
                            </button>
                            {item.title}
                        </li>
                    {/each}
                </ul>
            {/if}

            <form onsubmit="{add}">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                    <div class="join">
                        <input
                                type="text"
                                bind:value={newItem}
                                bind:this={input}
                                placeholder="new item "
                                class="input input-bordered join-item"
                                disabled={opLoading}
                        />

                        <button class="btn join-item" disabled={opLoading}>
                            {#if opLoading}
                                <span class="loading loading-spinner"></span>
                            {:else}
                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-5">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                                </svg>
                            {/if}
                        </button>
                    </div>

                    <button
                            type="button"
                            class="btn"
                            disabled={opLoading}
                            onclick={() => random()}
                    >
                        {#if opLoading}
                            <span class="loading loading-spinner"></span>
                        {/if}
                        Add items
                    </button>
                </div>

            </form>

        </div>
    </div>
</section>
<style>
    .item {
        vertical-align: middle;
    }

    .removing {
        opacity: 0.5;
    }
</style>