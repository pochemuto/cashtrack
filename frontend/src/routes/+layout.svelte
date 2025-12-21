<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";

    let {children} = $props();
</script>

<svelte:head>
    <link rel="icon" href={favicon}/>
</svelte:head>


<div class="navbar bg-base-100 shadow-sm">
    <div class="flex-1">
        <a class="btn btn-ghost text-xl">Cashtrack</a>
    </div>
</div>

<main class="mx-auto w-full max-w-6xl px-4 py-8">
    {@render children()}
</main>