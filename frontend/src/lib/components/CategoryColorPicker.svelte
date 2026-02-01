<script lang="ts">
    import {onMount} from "svelte";
    import ColorPicker from "svelte-awesome-color-picker";
    import CategoryBadge from "$lib/components/CategoryBadge.svelte";

    export let hex: string | null = null;
    export let label = "Цвет";
    export let popoverClass = "rounded-box border border-base-200 bg-base-100 p-3 shadow-xl";
    export let inline = false;
    export let nullable = true;

    let open = false;
    let trigger: HTMLButtonElement | null = null;
    let panel: HTMLDivElement | null = null;
    let popoverStyle = "";

    function updatePosition() {
        if (inline) {
            return;
        }
        if (!trigger) {
            return;
        }
        const rect = trigger.getBoundingClientRect();
        const popoverWidth = 320;
        const left = Math.min(rect.left, window.innerWidth - popoverWidth - 16);
        const top = rect.bottom + 8;
        popoverStyle = `position: fixed; left: ${Math.max(16, left)}px; top: ${top}px; z-index: 60;`;
    }

    function openPicker() {
        if (inline) {
            return;
        }
        open = true;
        updatePosition();
        requestAnimationFrame(updatePosition);
    }

    function closePicker() {
        if (inline) {
            return;
        }
        open = false;
    }

    function togglePicker() {
        if (inline) {
            return;
        }
        if (open) {
            closePicker();
        } else {
            openPicker();
        }
    }

    function handleDocumentClick(event: MouseEvent) {
        if (inline) {
            return;
        }
        if (!open) {
            return;
        }
        const target = event.target as Node | null;
        if (panel?.contains(target) || trigger?.contains(target)) {
            return;
        }
        closePicker();
    }

    function handleKeydown(event: KeyboardEvent) {
        if (inline) {
            return;
        }
        if (event.key === "Escape") {
            closePicker();
        }
    }

    onMount(() => {
        if (inline) {
            return;
        }
        const handleReposition = () => {
            if (open) {
                updatePosition();
            }
        };
        window.addEventListener("click", handleDocumentClick, true);
        window.addEventListener("keydown", handleKeydown);
        window.addEventListener("resize", handleReposition);
        window.addEventListener("scroll", handleReposition, true);
        return () => {
            window.removeEventListener("click", handleDocumentClick, true);
            window.removeEventListener("keydown", handleKeydown);
            window.removeEventListener("resize", handleReposition);
            window.removeEventListener("scroll", handleReposition, true);
        };
    });
</script>

{#if inline}
    <div class="max-w-[320px]">
        <ColorPicker
            bind:hex={hex}
            label={label}
            {nullable}
            isAlpha={false}
            isTextInput={false}
            isDialog={false}
        />
    </div>
{:else}
    <div class="flex items-center gap-2">
        <button
            class="p-0"
            type="button"
            aria-expanded={open}
            on:click={togglePicker}
            bind:this={trigger}
        >
            <CategoryBadge
                name={hex ?? label}
                color={hex ?? ""}
                primaryWhenNoColor={true}
                className="cursor-pointer"
            />
        </button>
    </div>

    {#if open}
        <div style={popoverStyle} class="max-w-[320px]">
            <div class={popoverClass} bind:this={panel}>
                <ColorPicker
                    bind:hex={hex}
                    label={label}
                    {nullable}
                    isAlpha={false}
                    isTextInput={false}
                    isDialog={false}
                />
            </div>
        </div>
    {/if}
{/if}
