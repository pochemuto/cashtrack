<script lang="ts">
    export let name: string;
    export let color = "";
    export let className = "";
    export let editable = false;
    export let placeholder = "Название";

    let badgeElement: HTMLSpanElement | null = null;

    $: normalizedColor = color?.trim() ?? "";
    $: hex = normalizedColor.startsWith("#") ? normalizedColor.slice(1) : normalizedColor;
    $: hasColor = hex.length === 6;
    $: badgeStyle = (() => {
        if (!hasColor) {
            return "";
        }
        const r = parseInt(hex.slice(0, 2), 16);
        const g = parseInt(hex.slice(2, 4), 16);
        const b = parseInt(hex.slice(4, 6), 16);
        const luminance = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
        const textColor = luminance > 0.6 ? "#000000" : "#FFFFFF";
        return `background-color: #${hex.toUpperCase()}; border-color: #${hex.toUpperCase()}; color: ${textColor};`;
    })();
    $: badgeClass = `badge badge-ghost${className ? ` ${className}` : ""}${editable ? " badge-editable" : ""}`;

    function handleKeydown(event: KeyboardEvent) {
        if (!editable) {
            return;
        }
        if (event.key === "Enter") {
            event.preventDefault();
            badgeElement?.blur();
        }
    }

    export function focus() {
        badgeElement?.focus();
    }
</script>

{#if editable}
    <span
        bind:this={badgeElement}
        class={badgeClass}
        style={badgeStyle}
        contenteditable="true"
        role="textbox"
        aria-label="Название категории"
        aria-multiline="false"
        tabindex="0"
        data-placeholder={placeholder}
        bind:textContent={name}
        on:keydown={handleKeydown}
    />
{:else}
    <span bind:this={badgeElement} class={badgeClass} style={badgeStyle}>
        {name}
    </span>
{/if}

<style>
    .badge-editable {
        cursor: text;
        min-width: 10rem;
        justify-content: flex-start;
        outline: none;
    }
    .badge-editable:focus-visible {
        outline: 2px solid oklch(var(--p));
        outline-offset: 2px;
    }
    .badge-editable:empty::before {
        content: attr(data-placeholder);
        opacity: 0.5;
    }
</style>
