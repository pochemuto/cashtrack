<script lang="ts">
    import {onMount} from "svelte";
    import {resolveApiUrl} from "$lib/url";
    import CategoryBadge from "$lib/components/CategoryBadge.svelte";
    import {user} from "../../user";

    type TransactionItem = {
        id: number;
        source_file_id: number;
        source_file_row: number;
        parser_name: string;
        posted_date: string;
        description: string;
        amount: string;
        currency: string;
        transaction_id: string;
        entry_type: string;
        source_account_number: string;
        source_card_number: string;
        created_at: string;
        category_id: number | null;
    };

    type TransactionSummary = {
        count: number;
        total: string;
        average: string;
        median: string;
        currency: string;
        unique_accounts: number;
        date_range_start: string;
        date_range_end: string;
    };

    type TransactionListResponse = {
        items: TransactionItem[];
        summary: TransactionSummary;
    };

    type SankeyChart = {
        data: Array<Record<string, unknown>>;
        layout: Record<string, unknown>;
        config: Record<string, unknown>;
    };

    type CategoryItem = {
        id: number;
        name: string;
        color: string | null;
        created_at: string;
    };

    let transactions: TransactionItem[] = [];
    let categories: CategoryItem[] = [];
    let summary: TransactionSummary | null = null;
    let loading = false;
    let categoriesLoading = false;
    let listError = "";
    let updateError = "";
    let categoryUpdates: Record<number, boolean> = {};
    let lastUserId: number | null = null;
    let plotlyLoading = false;
    let plotlyError = "";
    let plotly: any = null;
    let sankeyContainer: HTMLDivElement | null = null;
    let sankeyChart: SankeyChart | null = null;

    let fromDate = "";
    let toDate = "";
    let sourceFileId = "";
    let entryType = "";
    let searchText = "";
    let accountNumber = "";
    let cardNumber = "";
    let categoryFilter = "";
    let calendarRange = "";
    let calendarElement: HTMLElement & {value?: string} | null = null;
    let calendarOpen = false;
    let calendarPopover: HTMLElement | null = null;
    let calendarInput: HTMLInputElement | null = null;
    let calendarUpdating = false;
    let textFilterDebounceTimer: ReturnType<typeof setTimeout> | null = null;
    let lastAppliedSignature = "";
    let lastAppliedTextSignature = "";
    let lastAppliedNonTextSignature = "";
    const textFilterDebounceMs = 400;
    let textFilterSignature = "";
    let nonTextFilterSignature = "";
    const sankeySourceColor = "#e2e8f0";
    const sankeyCategoryFallbackColor = "#94a3b8";

    function formatYmd(date: Date): string {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, "0");
        const day = String(date.getDate()).padStart(2, "0");
        return `${year}-${month}-${day}`;
    }

    function formatDate(value: string): string {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, "0");
        const day = String(date.getDate()).padStart(2, "0");
        return `${year}-${month}-${day}`;
    }

    function amountWithSign(value: string, entryType: string): string {
        if (!value) {
            return "";
        }
        if (entryType === "credit" && !value.startsWith("+")) {
            return "+" + value;
        }
        return value;
    }

    function handleCalendarRangeChange(event: Event) {
        if (calendarUpdating) {
            return;
        }
        const target = event.currentTarget as HTMLInputElement;
        const value = target.value || "";
        if (!value) {
            fromDate = "";
            toDate = "";
            calendarOpen = false;
            return;
        }
        const [start, end] = value.split("/");
        fromDate = start ?? "";
        toDate = end ?? "";
        if (fromDate && toDate) {
            calendarOpen = false;
        }
    }

    function toggleCalendar() {
        calendarOpen = !calendarOpen;
    }

    function closeCalendar() {
        calendarOpen = false;
    }

    function applyPresetRange(label: string) {
        const today = new Date();
        today.setHours(0, 0, 0, 0);

        let start: Date;
        let end: Date;

        switch (label) {
            case "current-month": {
                const year = today.getFullYear();
                const month = today.getMonth();
                start = new Date(year, month, 1);
                end = new Date(year, month + 1, 0);
                break;
            }
            case "previous-month": {
                const year = today.getFullYear();
                const month = today.getMonth() - 1;
                start = new Date(year, month, 1);
                end = new Date(year, month + 1, 0);
                break;
            }
            case "current-year": {
                const year = today.getFullYear();
                start = new Date(year, 0, 1);
                end = new Date(year, 11, 31);
                break;
            }
            case "previous-year": {
                const year = today.getFullYear() - 1;
                start = new Date(year, 0, 1);
                end = new Date(year, 11, 31);
                break;
            }
            case "last-30-days": {
                end = new Date(today);
                start = new Date(today);
                start.setDate(start.getDate() - 29);
                break;
            }
            case "last-90-days": {
                end = new Date(today);
                start = new Date(today);
                start.setDate(start.getDate() - 89);
                break;
            }
            default:
                return;
        }

        calendarUpdating = true;
        fromDate = formatYmd(start);
        toDate = formatYmd(end);
        calendarOpen = false;
        queueMicrotask(() => {
            calendarUpdating = false;
        });
    }

    function formatSummaryAmount(value: string): string {
        const parsed = Number(value);
        if (!Number.isFinite(parsed)) {
            return value;
        }
        return parsed.toFixed(2);
    }

    function formatSummaryDateRange(start: string, end: string): string {
        if (!start && !end) {
            return "—";
        }
        if (start && end) {
            return `${start} — ${end}`;
        }
        return start || end;
    }

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

    function entryTypeLabel(entryType: string): string {
        if (entryType === "debit") {
            return "Debit";
        }
        if (entryType === "credit") {
            return "Credit";
        }
        return "Unknown";
    }

    function getCategoryDescriptor(categoryId: number | null, categoryLookup: Map<number, CategoryItem>) {
        if (categoryId === null) {
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

    function buildSankeyChart(items: TransactionItem[], categoryItems: CategoryItem[]): SankeyChart | null {
        if (!items.length) {
            return null;
        }

        const categoryLookup = new Map<number, CategoryItem>();
        for (const category of categoryItems) {
            categoryLookup.set(category.id, category);
        }

        const nodeIndex = new Map<string, number>();
        const nodeLabels: string[] = [];
        const nodeColors: string[] = [];
        const links = new Map<string, {source: number; target: number; value: number; color: string}>();
        const creditTotals = new Map<string, {label: string; color: string; value: number}>();
        const debitTotals = new Map<string, {label: string; color: string; value: number}>();
        const netIncomeLabel = "Net income";
        const remainderLabel = "Unknown";
        let totalCredits = 0;
        let totalDebits = 0;

        const ensureNode = (key: string, label: string, color: string) => {
            if (!nodeIndex.has(key)) {
                nodeIndex.set(key, nodeLabels.length);
                nodeLabels.push(label);
                nodeColors.push(color);
            }
            return nodeIndex.get(key) ?? 0;
        };

        const addTotal = (
            totals: Map<string, {label: string; color: string; value: number}>,
            key: string,
            label: string,
            color: string,
            value: number
        ) => {
            const existing = totals.get(key);
            if (existing) {
                existing.value += value;
            } else {
                totals.set(key, {label, color, value});
            }
        };

        for (const tx of items) {
            const amount = Number(tx.amount);
            if (!Number.isFinite(amount) || amount === 0) {
                continue;
            }
            const value = Math.abs(amount);
            const categoryInfo = getCategoryDescriptor(tx.category_id, categoryLookup);

            if (tx.entry_type === "credit") {
                totalCredits += value;
                addTotal(creditTotals, categoryInfo.key, categoryInfo.label, categoryInfo.color, value);
            } else if (tx.entry_type === "debit") {
                totalDebits += value;
                addTotal(debitTotals, categoryInfo.key, categoryInfo.label, categoryInfo.color, value);
            }
        }

        if (!creditTotals.size && !debitTotals.size) {
            return null;
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
            const linkColor = colorWithAlpha(entry.color, 0.45, entry.color);
            links.set(linkKey, {source: netIncomeIndex, target: targetIndex, value: entry.value, color: linkColor});
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
        }

        if (!links.size) {
            return null;
        }

        const sources: number[] = [];
        const targets: number[] = [];
        const values: number[] = [];
        const colors: string[] = [];

        for (const link of links.values()) {
            sources.push(link.source);
            targets.push(link.target);
            values.push(Number(link.value.toFixed(2)));
            colors.push(link.color);
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
                    },
                    link: {
                        source: sources,
                        target: targets,
                        value: values,
                        color: colors,
                        hovertemplate: "%{source.label} -> %{target.label}<br>%{value:.2f}<extra></extra>",
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

    function clearTextFilterDebounce() {
        if (textFilterDebounceTimer) {
            clearTimeout(textFilterDebounceTimer);
            textFilterDebounceTimer = null;
        }
    }

    function triggerFilterLoad() {
        const signature = `${nonTextFilterSignature}|${textFilterSignature}`;
        if (signature === lastAppliedSignature) {
            return;
        }
        lastAppliedSignature = signature;
        lastAppliedTextSignature = textFilterSignature;
        lastAppliedNonTextSignature = nonTextFilterSignature;
        void loadTransactions();
    }

    function scheduleTextFilterLoad() {
        clearTextFilterDebounce();
        const expectedSignature = textFilterSignature;
        textFilterDebounceTimer = setTimeout(() => {
            textFilterDebounceTimer = null;
            if (expectedSignature !== textFilterSignature) {
                return;
            }
            triggerFilterLoad();
        }, textFilterDebounceMs);
    }


    async function loadTransactions() {
        if (!$user || !$user.id) {
            transactions = [];
            summary = null;
            listError = "";
            loading = false;
            return;
        }

        loading = true;
        listError = "";

        const params = new URLSearchParams();
        if (fromDate) params.set("from", fromDate);
        if (toDate) params.set("to", toDate);
        if (sourceFileId) params.set("source_file_id", sourceFileId);
        if (entryType) params.set("entry_type", entryType);
        if (searchText) params.set("search", searchText);
        if (categoryFilter) params.set("category_id", categoryFilter);
        if (accountNumber) params.set("account_number", accountNumber);
        if (cardNumber) params.set("card_number", cardNumber);

        try {
            const response = await fetch(resolveApiUrl(`api/transactions?${params.toString()}`), {
                credentials: "include",
            });
            if (!response.ok) {
                if (response.status === 401) {
                    listError = "Нужен вход для просмотра транзакций.";
                    transactions = [];
                    summary = null;
                    return;
                }
                listError = "Не удалось загрузить транзакции.";
                summary = null;
                return;
            }
            const payload = (await response.json()) as TransactionListResponse;
            transactions = payload.items ?? [];
            summary = payload.summary ?? null;
        } catch {
            listError = "Не удалось загрузить транзакции.";
            summary = null;
        } finally {
            loading = false;
        }
    }

    async function loadCategories() {
        if (!$user || !$user.id) {
            categories = [];
            categoriesLoading = false;
            return;
        }

        categoriesLoading = true;
        try {
            const response = await fetch(resolveApiUrl("api/categories"), {credentials: "include"});
            if (!response.ok) {
                categories = [];
                return;
            }
            categories = (await response.json()) as CategoryItem[];
        } finally {
            categoriesLoading = false;
        }
    }

    async function updateTransactionCategory(transactionId: number, categoryId: number | null) {
        updateError = "";
        categoryUpdates = {...categoryUpdates, [transactionId]: true};
        try {
            const response = await fetch(resolveApiUrl(`api/transactions/${transactionId}/category`), {
                method: "PATCH",
                headers: {"Content-Type": "application/json"},
                credentials: "include",
                body: JSON.stringify({category_id: categoryId}),
            });
            if (!response.ok) {
                updateError = "Не удалось сохранить категорию.";
                return;
            }
            transactions = transactions.map((tx) =>
                tx.id === transactionId ? {...tx, category_id: categoryId} : tx
            );
        } catch {
            updateError = "Не удалось сохранить категорию.";
        } finally {
            const {[transactionId]: _removed, ...rest} = categoryUpdates;
            categoryUpdates = rest;
        }
    }

    function closeCategoryMenu(event: MouseEvent) {
        const target = event.currentTarget as HTMLElement | null;
        if (!target) {
            return;
        }
        requestAnimationFrame(() => {
            target.blur();
            const dropdown = target.closest(".dropdown");
            const toggle = dropdown?.querySelector("button");
            if (toggle instanceof HTMLElement) {
                toggle.blur();
            }
        });
    }

    function handleCategorySelect(event: MouseEvent, transactionId: number, categoryId: number | null) {
        closeCategoryMenu(event);
        void updateTransactionCategory(transactionId, categoryId);
    }

    function resetFilters() {
        fromDate = "";
        toDate = "";
        sourceFileId = "";
        entryType = "";
        searchText = "";
        categoryFilter = "";
        accountNumber = "";
        cardNumber = "";
        clearTextFilterDebounce();
        triggerFilterLoad();
    }

    $: if ($user?.id && $user.id !== lastUserId) {
        lastUserId = $user.id;
        clearTextFilterDebounce();
        lastAppliedSignature = "";
        lastAppliedTextSignature = "";
        lastAppliedNonTextSignature = "";
        triggerFilterLoad();
        void loadCategories();
    }

    $: if ($user?.id) {
        const nextSignature = `${nonTextFilterSignature}|${textFilterSignature}`;
        if (nextSignature !== lastAppliedSignature) {
            const nonTextChanged = nonTextFilterSignature !== lastAppliedNonTextSignature;
            if (nonTextChanged) {
                clearTextFilterDebounce();
                triggerFilterLoad();
            } else {
                scheduleTextFilterLoad();
            }
        }
    }

    $: textFilterSignature = [searchText, sourceFileId, accountNumber, cardNumber].join("|");

    $: nonTextFilterSignature = [fromDate, toDate, entryType, categoryFilter].join("|");

    $: sankeyChart = buildSankeyChart(transactions, categories);

    $: if (plotly && sankeyContainer) {
        if (sankeyChart) {
            plotly.react(sankeyContainer, sankeyChart.data, sankeyChart.layout, sankeyChart.config);
        } else {
            plotly.purge(sankeyContainer);
        }
    }

    $: if (fromDate && toDate) {
        const next = `${fromDate}/${toDate}`;
        if (calendarRange !== next) {
            calendarRange = next;
        }
    } else if (!fromDate && !toDate && calendarRange) {
        calendarRange = "";
    }

    $: if (!calendarUpdating && calendarElement && calendarElement.value !== calendarRange) {
        calendarElement.value = calendarRange;
    }

    $: calendarInputValue =
        fromDate && toDate ? `${fromDate} — ${toDate}` : fromDate || toDate ? `${fromDate || "—"} — ${toDate || "—"}` : "";

    onMount(async () => {
        await import("cally");

        const handleDocumentClick = (event: MouseEvent) => {
            if (!calendarOpen) {
                return;
            }
            const target = event.target as Node | null;
            if (calendarPopover && target && calendarPopover.contains(target)) {
                return;
            }
            if (calendarInput && target && calendarInput.contains(target)) {
                return;
            }
            calendarOpen = false;
        };

        const handleKeydown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                calendarOpen = false;
            }
        };

        document.addEventListener("click", handleDocumentClick, true);
        window.addEventListener("keydown", handleKeydown);

        return () => {
            document.removeEventListener("click", handleDocumentClick, true);
            window.removeEventListener("keydown", handleKeydown);
        };
    });

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

<svelte:head>
    <title>Transactions</title>
</svelte:head>

<section class="mx-auto w-full max-w-none">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Транзакции</h1>
            </div>

            <div class="grid gap-4 lg:grid-cols-2">
                <div class="form-control">
                    <label class="label" for="date-range">
                        <span class="label-text">Диапазон дат</span>
                    </label>
                    <div class="relative">
                        <input
                            class="input input-bordered w-full"
                            type="text"
                            id="date-range"
                            readonly
                            placeholder="Выберите диапазон"
                            bind:this={calendarInput}
                            value={calendarInputValue}
                            on:click={toggleCalendar}
                        />
                        {#if calendarOpen}
                            <div
                                class="absolute z-20 mt-2 rounded-box border border-base-200 bg-base-100 p-2 shadow"
                                bind:this={calendarPopover}
                            >
                                <div class="mb-2 grid gap-2 sm:grid-cols-2">
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("current-month")}>
                                        Текущий месяц
                                    </button>
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("previous-month")}>
                                        Предыдущий месяц
                                    </button>
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("current-year")}>
                                        Текущий год
                                    </button>
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("previous-year")}>
                                        Предыдущий год
                                    </button>
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("last-30-days")}>
                                        Последние 30 дней
                                    </button>
                                    <button class="btn btn-ghost btn-xs justify-start" type="button" on:click={() => applyPresetRange("last-90-days")}>
                                        Последние 90 дней
                                    </button>
                                </div>
                                <calendar-range
                                    class="cally"
                                    bind:this={calendarElement}
                                    value={calendarRange}
                                    on:change={handleCalendarRangeChange}
                                >
                                    <calendar-month></calendar-month>
                                </calendar-range>
                                <div class="mt-2 text-xs opacity-70">
                                    {fromDate || "—"} — {toDate || "—"}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>
                <div class="form-control flex flex-col">
                    <label class="label" for="category-filter">
                        <span class="label-text">Категория</span>
                    </label>
                    <select class="select select-bordered" id="category-filter" bind:value={categoryFilter} disabled={categoriesLoading}>
                        <option value="">Все</option>
                        {#each categories as category}
                            <option value={String(category.id)}>{category.name}</option>
                        {/each}
                    </select>
                </div>
                <div class="form-control flex flex-col">
                    <label class="label" for="search-text">
                        <span class="label-text">Поиск по описанию</span>
                    </label>
                    <input class="input input-bordered w-full" type="text" id="search-text" bind:value={searchText} placeholder="например Uber" />
                </div>
                <div class="form-control flex flex-col">
                    <label class="label" for="entry-type">
                        <span class="label-text">Тип списания</span>
                    </label>
                    <select class="select select-bordered" id="entry-type" bind:value={entryType}>
                        <option value="">Все</option>
                        <option value="debit">Debit</option>
                        <option value="credit">Credit</option>
                    </select>
                </div>
            </div>

            <details class="collapse collapse-arrow border border-base-200 bg-base-100">
                <summary class="collapse-title text-sm font-medium">Расширенные фильтры</summary>
                <div class="collapse-content">
                    <div class="grid gap-4 lg:grid-cols-3">
                        <div class="form-control flex flex-col">
                            <label class="label" for="source-file-id">
                                <span class="label-text">Источник (ID файла)</span>
                            </label>
                            <input class="input input-bordered" type="text" id="source-file-id" bind:value={sourceFileId} placeholder="например 12" />
                        </div>
                        <div class="form-control flex flex-col">
                            <label class="label" for="account-number">
                                <span class="label-text">Номер счета</span>
                            </label>
                            <input class="input input-bordered" type="text" id="account-number" bind:value={accountNumber} />
                        </div>
                        <div class="form-control flex flex-col">
                            <label class="label" for="card-number">
                                <span class="label-text">Номер карты</span>
                            </label>
                            <input class="input input-bordered" type="text" id="card-number" bind:value={cardNumber} />
                        </div>
                    </div>
                </div>
            </details>

            <div class="flex flex-wrap justify-end gap-3">
                <button class="btn btn-ghost" type="button" on:click={resetFilters} disabled={loading}>
                    Сбросить фильтры
                </button>
            </div>

            {#if listError}
                <div class="alert alert-error">
                    <span>{listError}</span>
                </div>
            {/if}
            {#if updateError}
                <div class="alert alert-error">
                    <span>{updateError}</span>
                </div>
            {/if}

            {#if loading}
                <div class="text-sm opacity-70">Загрузка транзакций...</div>
            {:else if transactions.length === 0}
                <div class="text-sm opacity-70">Нет транзакций по выбранным фильтрам.</div>
            {:else}
                {#if summary}
                    <div class="stats stats-vertical lg:stats-horizontal bg-base-100 shadow">
                        <div class="stat">
                            <div class="stat-title">Количество</div>
                            <div class="stat-value text-lg">{summary.count}</div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Общая сумма</div>
                            <div class="stat-value text-lg">
                                {formatSummaryAmount(summary.total)} {summary.currency}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Средняя сумма</div>
                            <div class="stat-value text-lg">
                                {formatSummaryAmount(summary.average)} {summary.currency}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Медианная сумма</div>
                            <div class="stat-value text-lg">
                                {formatSummaryAmount(summary.median)} {summary.currency}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Уникальные счета</div>
                            <div class="stat-value text-lg">{summary.unique_accounts}</div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Диапазон дат</div>
                            <div class="stat-value text-lg">
                                {formatSummaryDateRange(summary.date_range_start, summary.date_range_end)}
                            </div>
                        </div>
                    </div>
                {/if}
                <details class="collapse collapse-arrow border border-base-200 bg-base-100" on:toggle={handleSankeyToggle}>
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
                <div class="overflow-x-auto">
                    <table class="table">
                        <thead>
                        <tr>
                            <th>Дата</th>
                            <th>Описание</th>
                            <th>Категория</th>
                            <th class="text-right">Сумма</th>
                            <th>Валюта</th>
                        </tr>
                        </thead>
                        <tbody>
                        {#each transactions as tx}
                            <tr>
                                <td class="whitespace-nowrap">{formatDate(tx.posted_date)}</td>
                                <td>
                                    <div class="font-medium">{tx.description}</div>
                                    <div class="text-xs opacity-70">ID: {tx.transaction_id || "—"}</div>
                                </td>
                                <td class="whitespace-nowrap">
                                    <div class="dropdown dropdown-start">
                                        <button
                                            class="p-0"
                                            type="button"
                                            disabled={categoriesLoading || !categories.length || categoryUpdates[tx.id]}
                                        >
                                            {#if tx.category_id}
                                                <CategoryBadge
                                                    name={categories.find((category) => category.id === tx.category_id)?.name || "Категория"}
                                                    color={categories.find((category) => category.id === tx.category_id)?.color || ""}
                                                    primaryWhenNoColor={true}
                                                />
                                            {:else}
                                                <CategoryBadge name="Без категории" />
                                            {/if}
                                        </button>
                                        <ul class="menu dropdown-content z-20 mt-2 w-56 rounded-box bg-base-100 p-2 shadow">
                                            <li>
                                                <button type="button" on:click={(event) => handleCategorySelect(event, tx.id, null)}>
                                                    <CategoryBadge name="Без категории" />
                                                </button>
                                            </li>
                                            {#each categories as category}
                                                <li>
                                                    <button type="button" on:click={(event) => handleCategorySelect(event, tx.id, category.id)}>
                                                        <CategoryBadge name={category.name} color={category.color} primaryWhenNoColor={true} />
                                                    </button>
                                                </li>
                                            {/each}
                                        </ul>
                                    </div>
                                </td>
                                <td class="text-right font-medium">{amountWithSign(tx.amount, tx.entry_type)}</td>
                                <td>{tx.currency}</td>
                            </tr>
                        {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    </div>
</section>

<style>
</style>
