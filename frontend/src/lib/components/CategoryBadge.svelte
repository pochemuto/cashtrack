<script lang="ts">
    export let name: string;
    export let color = "";
    export let primaryWhenNoColor = false;
    export let className = "";

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
    $: badgeClass = `badge badge-ghost${!hasColor && primaryWhenNoColor ? " badge-primary" : ""}${className ? ` ${className}` : ""}`;
</script>

<span class={badgeClass} style={badgeStyle}>
    {name}
</span>
