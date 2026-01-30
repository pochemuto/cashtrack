<script lang="ts">
    import {onMount} from "svelte";
    import type {Category} from "$lib/gen/api/v1/categories_pb";
    import type {Transaction} from "$lib/gen/api/v1/transactions_pb";
    import {persistedBoolean} from "$lib/stores/persistedBoolean";
    import {centsToNumber, formatChfAmount} from "$lib/money";

    type SankeyChart = {
        data: Array<Record<string, unknown>>;
        layout: Record<string, unknown>;
        config: Record<string, unknown>;
    };

    export let transactions: Transaction[] = [];
    export let categories: Category[] = [];

    const sankeyOpen = persistedBoolean("transactions.sankey.open", false);

    let plotlyLoading = false;
    let plotlyError = "";
    let plotly: any = null;
    let sankeyContainer: HTMLDivElement | null = null;
    let sankeyChart: SankeyChart | null = null;

    const sankeySourceColor = "#e2e8f0";
    const sankeyCategoryFallbackColor = "#94a3b8";

    function normalizeColor(value: string | null, fallback: string): string {
        if (!value) {
            return fallback;
        }
        const trimmed = value.trim();
        return trimmed ? trimmed : fallback;
    }

    function colorWithAlpha(value: string, alpha: number, fallback: string): string {
        const trimmed = value.trim();
        if (trimmed.startsWith("#")) {
            const hex = trimmed.slice(1);
            if (hex.length === 3) {
                const r = Number.parseInt(hex[0] + hex[0], 16);
                const g = Number.parseInt(hex[1] + hex[1], 16);
                const b = Number.parseInt(hex[2] + hex[2], 16);
                return `rgba(${r}, ${g}, ${b}, ${alpha})`;
            }
            if (hex.length === 6) {
                const r = Number.parseInt(hex.slice(0, 2), 16);
                const g = Number.parseInt(hex.slice(2, 4), 16);
                const b = Number.parseInt(hex.slice(4, 6), 16);
                return `rgba(${r}, ${g}, ${b}, ${alpha})`;
            }
        }
        if (trimmed.startsWith("rgb(")) {
            const parts = trimmed.slice(4, -1).split(",").map((item) => item.trim());
            if (parts.length >= 3) {
                return `rgba(${parts[0]}, ${parts[1]}, ${parts[2]}, ${alpha})`;
            }
        }
        if (trimmed.startsWith("rgba(")) {
            const parts = trimmed.slice(5, -1).split(",").map((item) => item.trim());
            if (parts.length >= 3) {
                return `rgba(${parts[0]}, ${parts[1]}, ${parts[2]}, ${alpha})`;
            }
        }
        return fallback;
    }


    function getCategoryDescriptor(categoryId: number | undefined, categoryLookup: Map<number, Category>) {
        if (categoryId === undefined) {
            return {
                key: "none",
                label: "Без категории",
                color: sankeyCategoryFallbackColor,
            };
        }
        const category = categoryLookup.get(categoryId);
        if (!category) {
            return {
                key: String(categoryId),
                label: `Категория ${categoryId}`,
                color: sankeyCategoryFallbackColor,
            };
        }
        return {
            key: String(category.id),
            label: category.name || `Категория ${category.id}`,
            color: normalizeColor(category.color, sankeyCategoryFallbackColor),
        };
    }

    function buildSankeyChart(items: Transaction[], categoryItems: Category[]): SankeyChart | null {
        if (!items.length) {
            return null;
        }

        const categoryLookup = new Map<number, Category>();
        for (const category of categoryItems) {
            categoryLookup.set(category.id, category);
        }

        const nodeIndex = new Map<string, number>();
        const nodeLabels: string[] = [];
        const nodeColors: string[] = [];
        const links = new Map<string, {source: number; target: number; value: number; color: string}>();
        const creditTotals = new Map<string, {label: string; color: string; value: number; count: number}>();
        const debitTotals = new Map<string, {label: string; color: string; value: number; count: number}>();
        const netIncomeLabel = "Net income";
        const remainderLabel = "Unknown";
        let totalCredits = 0;
        let totalDebits = 0;
        let totalCreditCount = 0;

        const ensureNode = (key: string, label: string, color: string) => {
            if (!nodeIndex.has(key)) {
                nodeIndex.set(key, nodeLabels.length);
                nodeLabels.push(label);
                nodeColors.push(color);
            }
            return nodeIndex.get(key) ?? 0;
        };

        const addTotal = (
            totals: Map<string, {label: string; color: string; value: number; count: number}>,
            key: string,
            label: string,
            color: string,
            value: number,
            count: number
        ) => {
            const existing = totals.get(key);
            if (existing) {
                existing.value += value;
                existing.count += count;
            } else {
                totals.set(key, {label, color, value, count});
            }
        };

        for (const tx of items) {
            const amount = centsToNumber(tx.amount);
            if (!Number.isFinite(amount) || amount === 0) {
                continue;
            }
            const value = Math.abs(amount);
            const categoryInfo = getCategoryDescriptor(tx.categoryId, categoryLookup);

            if (tx.entryType === "credit") {
                totalCredits += value;
                totalCreditCount += 1;
                addTotal(creditTotals, categoryInfo.key, categoryInfo.label, categoryInfo.color, value, 1);
            } else if (tx.entryType === "debit") {
                totalDebits += value;
                addTotal(debitTotals, categoryInfo.key, categoryInfo.label, categoryInfo.color, value, 1);
            }
        }

        if (!creditTotals.size && !debitTotals.size) {
            return null;
        }

        const nodeStats = new Map<string, {count: number; amount: number}>();
        nodeStats.set("net:income", {count: totalCreditCount, amount: totalCredits});
        for (const [key, entry] of creditTotals.entries()) {
            nodeStats.set(`credit:${key}`, {count: entry.count, amount: entry.value});
        }
        for (const [key, entry] of debitTotals.entries()) {
            nodeStats.set(`debit:${key}`, {count: entry.count, amount: entry.value});
        }

        const netIncomeIndex = ensureNode("net:income", netIncomeLabel, sankeySourceColor);

        for (const [key, entry] of creditTotals.entries()) {
            const sourceIndex = ensureNode(`credit:${key}`, entry.label, entry.color);
            const linkKey = `${sourceIndex}:${netIncomeIndex}`;
            const linkColor = colorWithAlpha(entry.color, 0.45, entry.color);
            links.set(linkKey, {source: sourceIndex, target: netIncomeIndex, value: entry.value, color: linkColor});
        }

        for (const [key, entry] of debitTotals.entries()) {
            const targetIndex = ensureNode(`debit:${key}`, entry.label, entry.color);
            const linkKey = `${netIncomeIndex}:${targetIndex}`;
            links.set(linkKey, {source: netIncomeIndex, target: targetIndex, value: entry.value, color: entry.color});
        }

        const remainder = Number((totalCredits - totalDebits).toFixed(2));
        if (remainder > 0) {
            const remainderIndex = ensureNode(`debit:${remainderLabel}`, remainderLabel, sankeyCategoryFallbackColor);
            const linkKey = `${netIncomeIndex}:${remainderIndex}`;
            links.set(linkKey, {
                source: netIncomeIndex,
                target: remainderIndex,
                value: remainder,
                color: colorWithAlpha(sankeyCategoryFallbackColor, 0.45, sankeyCategoryFallbackColor),
            });
            nodeStats.set(`debit:${remainderLabel}`, {count: 0, amount: remainder});
        }

        if (!links.size) {
            return null;
        }

        const sources: number[] = [];
        const targets: number[] = [];
        const values: number[] = [];
        const colors: string[] = [];
        const linkCustomData: string[] = [];
        const nodeCustomData: Array<[number, string]> = nodeLabels.map(() => [0, formatChfAmount(0)]);

        for (const [key, stats] of nodeStats.entries()) {
            const index = nodeIndex.get(key);
            if (index !== undefined) {
                nodeCustomData[index] = [stats.count, formatChfAmount(stats.amount)];
            }
        }

        for (const link of links.values()) {
            sources.push(link.source);
            targets.push(link.target);
            values.push(Number(link.value.toFixed(2)));
            colors.push(link.color);
            linkCustomData.push(formatChfAmount(link.value));
        }

        const height = Math.min(640, Math.max(280, nodeLabels.length * 24));

        return {
            data: [
                {
                    type: "sankey",
                    orientation: "h",
                    node: {
                        pad: 18,
                        thickness: 16,
                        line: {color: "rgba(0,0,0,0.2)", width: 0.5},
                        label: nodeLabels,
                        color: nodeColors,
                        customdata: nodeCustomData,
                        hovertemplate:
                            "%{label}<br>Входящие транзакции: %{customdata[0]}<br>%{customdata[1]}<extra></extra>",
                    },
                    link: {
                        source: sources,
                        target: targets,
                        value: values,
                        color: colors,
                        customdata: linkCustomData,
                        hovertemplate: "%{source.label} → %{target.label}<br>%{customdata}<extra></extra>",
                    },
                },
            ],
            layout: {
                margin: {l: 10, r: 10, t: 10, b: 10},
                height,
                paper_bgcolor: "transparent",
                plot_bgcolor: "transparent",
            },
            config: {
                displayModeBar: false,
                responsive: true,
            },
        };
    }

    function handleSankeyToggle(event: Event) {
        const details = event.currentTarget as HTMLDetailsElement | null;
        if (!details || !details.open || !plotly || !sankeyContainer) {
            return;
        }
        requestAnimationFrame(() => {
            plotly.Plots.resize(sankeyContainer);
        });
    }

    $: sankeyChart = buildSankeyChart(transactions, categories);

    $: if (plotly && sankeyContainer) {
        if (sankeyChart) {
            plotly.react(sankeyContainer, sankeyChart.data, sankeyChart.layout, sankeyChart.config);
        } else {
            plotly.purge(sankeyContainer);
        }
    }

    onMount(() => {
        let cancelled = false;
        plotlyLoading = true;
        import("plotly.js-dist-min")
            .then((module) => {
                if (cancelled) {
                    return;
                }
                plotly = module.default ?? module;
            })
            .catch(() => {
                if (!cancelled) {
                    plotlyError = "Не удалось загрузить диаграмму.";
                }
            })
            .finally(() => {
                if (!cancelled) {
                    plotlyLoading = false;
                }
            });

        return () => {
            cancelled = true;
        };
    });
</script>

<details class="collapse collapse-arrow border border-base-200 bg-base-100" bind:open={$sankeyOpen} on:toggle={handleSankeyToggle}>
    <summary class="collapse-title text-sm font-medium">Sankey-диаграмма</summary>
    <div class="collapse-content">
        {#if plotlyError}
            <div class="text-sm text-error">{plotlyError}</div>
        {:else if plotlyLoading}
            <div class="text-sm opacity-70">Загрузка диаграммы...</div>
        {:else if !sankeyChart}
            <div class="text-sm opacity-70">Недостаточно данных для диаграммы.</div>
        {:else}
            <div class="min-h-[280px] w-full" bind:this={sankeyContainer}></div>
        {/if}
    </div>
</details>
