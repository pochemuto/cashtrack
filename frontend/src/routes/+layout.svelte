<script lang="ts">
    import { onMount } from "svelte";
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";

    let {children} = $props();

    const GOOGLE_CLIENT_ID = "1010772966942-khflv7f816n0bqebf7mll7hb0eu589r0.apps.googleusercontent.com";
    let googleButtonEl: HTMLDivElement | null = null;

    declare global {
        interface Window {
            google?: typeof google;
        }
    }

    function handleGoogleCredential(response: google.accounts.id.CredentialResponse) {
        console.info("Google credential received", response);
    }

    function initializeGoogleSignIn() {
        const googleApi = window.google;
        if (!googleApi?.accounts?.id) {
            return false;
        }

        googleApi.accounts.id.initialize({
            client_id: GOOGLE_CLIENT_ID,
            callback: handleGoogleCredential,
            auto_select: false
        });

        if (googleButtonEl) {
            googleApi.accounts.id.renderButton(googleButtonEl, {
                type: "standard",
                theme: "outline",
                size: "large",
                text: "signin_with",
                shape: "pill"
            });
        }

        googleApi.accounts.id.prompt();
        return true;
    }

    onMount(() => {
        let cancelled = false;

        const tryInit = () => {
            if (cancelled) {
                return;
            }
            if (initializeGoogleSignIn()) {
                return;
            }
            requestAnimationFrame(tryInit);
        };

        tryInit();

        return () => {
            cancelled = true;
            window.google?.accounts?.id?.cancel?.();
        };
    });
</script>

<svelte:head>
    <link rel="icon" href={favicon}/>
    <script src="https://accounts.google.com/gsi/client" async defer></script>
</svelte:head>


<div class="navbar bg-base-100 shadow-sm">
    <div class="flex-1">
        <a class="btn btn-ghost text-xl" href="/">Cashtrack</a>
    </div>
    <div class="flex-none">
        <div bind:this={googleButtonEl} class="min-h-[40px]"></div>
    </div>
</div>

<main class="mx-auto w-full max-w-6xl px-4 py-8">
    {@render children()}
</main>
