<script lang="ts">
    import { Client } from "../client";

    let count = $state(0);
    let message = $state("Nothing");
    async function greet() {
        count += 1;
        let res = await Client.greet({"name": "World"});
        message = res.greeting;
    }
</script>

<button onclick={greet}>Inc</button> {count}

<p>{message}</p>
<style>
    button {
        font-size: 2em;
    }
</style>