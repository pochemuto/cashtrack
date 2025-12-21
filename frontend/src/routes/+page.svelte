<script lang="ts">
    import {Todo} from "$lib/api";
    import type {ListItem} from "$lib/gen/api/v1/todo_pb"
    import {onMount} from "svelte";

    let result = $state<Promise<ListItem[]>>();
    let newItem = $state("");
    let input: HTMLInputElement;

    onMount(load);

    async function load() {
        result = Todo.list({}).then((response) => response.items);
    }

    async function remove(id: number) {
        let response = Todo.remove({id});
        result = Promise.resolve((await response).items);
    }

    async function add() {
        if (!newItem) {
            return;
        }
        let response = await Todo.add({items: [{title: newItem}]});
        result = Promise.resolve(response.items);
        newItem = "";
        input.focus();
    }
</script>

<svelte:head>
    <title>Todo</title>
</svelte:head>


<section class="mx-auto w-full max-w-2xl">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-4">


{#if result}
    {#await result}
        <p>Loading...</p>
    {:then value}
        <ul>
            {#each value as item}
                <li>
                    <button class="btn btn-sm remove-button" onclick={() => remove(item.id)}>x</button> {item.title}</li>
            {/each}
        </ul>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}

            <form onsubmit="{add}">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                <label class="form-control w-full">
                    <input
                            type="text"
                            bind:value={newItem}
                            bind:this={input}
                            placeholder="new item"
                            class="input input-bordered w-full"
                    />
                </label>

                <button class="btn">
                    +
                </button>
            </div>

            </form>

            </div>
    </div>
</section>  
<style>
    .remove-button {
        padding: 0 5px
    }
</style>