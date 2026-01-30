<script lang="ts">
    import {onMount} from "svelte";
    import {Transactions} from "$lib/api";
    import type {Transaction, TransactionSummary} from "$lib/gen/api/v1/transactions_pb";
    import {Code, ConnectError} from "@connectrpc/connect";
    import CategoryBadge from "$lib/components/CategoryBadge.svelte";
    import TransactionsSankey from "$lib/components/TransactionsSankey.svelte";
    import {categories, categoriesLoading, loadCategories} from "$lib/stores/categories";
    import {centsToNumber, formatCurrencyAmount, formatSignedCents} from "$lib/money";
    import {persistedBoolean} from "$lib/stores/persistedBoolean";
    import {user} from "../../user";

    type SankeyFilter = {
        kind: "node" | "link";
        entryType: "credit" | "debit";
        categoryId: number | null;
        label: string;
    };

    let transactions: Transaction[] = [];
    let summary: TransactionSummary | null = null;
    let loading = false;
    let listError = "";
    let updateError = "";
    let categoryUpdates: Record<number, boolean> = {};
    let lastUserId: number | null = null;

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
    let sankeyFilter: SankeyFilter | null = null;
    let tableTransactions: Transaction[] = [];
    let textFilterDebounceTimer: ReturnType<typeof setTimeout> | null = null;
    let lastAppliedSignature = "";
    let lastAppliedTextSignature = "";
    let lastAppliedNonTextSignature = "";
    const textFilterDebounceMs = 400;
    let textFilterSignature = "";
    let nonTextFilterSignature = "";
    const advancedFiltersOpen = persistedBoolean("transactions.advancedFilters.open", false);

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


    function formatSummaryDateRange(start: string, end: string): string {
        if (!start && !end) {
            return "—";
        }
        if (start && end) {
            return `${start} — ${end}`;
        }
        return start || end;
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

        try {
            const response = await Transactions.listTransactions({
                fromDate,
                toDate,
                sourceFileId: sourceFileId ? Number(sourceFileId) : 0,
                entryType,
                searchText,
                categoryId: categoryFilter ? Number(categoryFilter) : 0,
                accountNumber,
                cardNumber,
            });
            transactions = response.items ?? [];
            summary = response.summary ?? null;
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                listError = "Нужен вход для просмотра транзакций.";
                transactions = [];
                summary = null;
                return;
            }
            listError = "Не удалось загрузить транзакции.";
            summary = null;
        } finally {
            loading = false;
        }
    }

    async function updateTransactionCategory(transactionId: number, categoryId: number | null) {
        updateError = "";
        categoryUpdates = {...categoryUpdates, [transactionId]: true};
        try {
            const request: {transactionId: number; categoryId?: number} = {transactionId};
            if (categoryId !== null) {
                request.categoryId = categoryId;
            }
            await Transactions.updateTransactionCategory(request);
            transactions = transactions.map((tx) =>
                tx.id === transactionId ? {...tx, categoryId: categoryId ?? undefined} : tx
            );
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                updateError = "Нужен вход для изменения категории.";
                return;
            }
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

    function handleSankeyFilterChange(event: CustomEvent<SankeyFilter | null>) {
        sankeyFilter = event.detail;
    }

    function resetSankeyFilter() {
        sankeyFilter = null;
    }

    function matchesSankeyFilter(transaction: Transaction, filter: SankeyFilter): boolean {
        if (transaction.entryType !== filter.entryType) {
            return false;
        }
        if (filter.categoryId === null) {
            return transaction.categoryId === undefined || transaction.categoryId === null;
        }
        return transaction.categoryId === filter.categoryId;
    }

    function resolveSankeyCategoryLabel(filter: SankeyFilter): string {
        if (filter.label) {
            return filter.label;
        }
        if (filter.categoryId === null) {
            return "Без категории";
        }
        return $categories.find((category) => category.id === filter.categoryId)?.name || `Категория ${filter.categoryId}`;
    }

    function formatSankeyFilterLabel(filter: SankeyFilter): string {
        const entryLabel = filter.entryType === "credit" ? "Credit" : "Debit";
        const categoryLabel = resolveSankeyCategoryLabel(filter);
        return `${entryLabel}: ${categoryLabel}`;
    }

    function formatSankeyFilterType(filter: SankeyFilter): string {
        return filter.kind === "link" ? "Связь" : "Узел";
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

    $: tableTransactions = sankeyFilter ? transactions.filter((tx) => matchesSankeyFilter(tx, sankeyFilter)) : transactions;


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

    onMount(() => {
        void import("cally");

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
                    <select class="select select-bordered" id="category-filter" bind:value={categoryFilter} disabled={$categoriesLoading}>
                        <option value="">Все</option>
                        {#each $categories as category}
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

            <details class="collapse collapse-arrow border border-base-200 bg-base-100" bind:open={$advancedFiltersOpen}>
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
                                {formatCurrencyAmount(centsToNumber(summary.total), summary.currency)}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Средняя сумма</div>
                            <div class="stat-value text-lg">
                                {formatCurrencyAmount(centsToNumber(summary.average), summary.currency)}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Медианная сумма</div>
                            <div class="stat-value text-lg">
                                {formatCurrencyAmount(centsToNumber(summary.median), summary.currency)}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Уникальные счета</div>
                                <div class="stat-value text-lg">{summary.uniqueAccounts}</div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Диапазон дат</div>
                            <div class="stat-value text-lg">
                                {formatSummaryDateRange(summary.dateRangeStart, summary.dateRangeEnd)}
                            </div>
                        </div>
                    </div>
                {/if}
                <TransactionsSankey {transactions} categories={$categories} on:filterChange={handleSankeyFilterChange} />
                {#if sankeyFilter}
                    <div class="flex flex-wrap items-center justify-between gap-3 rounded-box border border-base-200 bg-base-100 p-3 text-sm">
                        <div class="flex flex-wrap items-center gap-2">
                            <span class="font-medium">Подфильтровано по Sankey:</span>
                            <span class="badge badge-outline">{formatSankeyFilterType(sankeyFilter)}</span>
                            <span>{formatSankeyFilterLabel(sankeyFilter)}</span>
                            <button class="btn btn-ghost btn-xs" type="button" on:click={resetSankeyFilter}>
                                Сбросить
                            </button>
                        </div>
                    </div>
                {/if}
                {#if tableTransactions.length === 0}
                    <div class="text-sm opacity-70">Нет транзакций для выбранного элемента Sankey.</div>
                {:else}
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
                            {#each tableTransactions as tx}
                                <tr>
                                    <td class="whitespace-nowrap">{formatDate(tx.postedDate)}</td>
                                    <td>
                                        <div class="font-medium">{tx.description}</div>
                                        <div class="text-xs opacity-70">ID: {tx.transactionId || "—"}</div>
                                    </td>
                                    <td class="whitespace-nowrap">
                                        <div class="dropdown dropdown-start">
                                            <button
                                                class="p-0"
                                                type="button"
                                                disabled={$categoriesLoading || !$categories.length || categoryUpdates[tx.id]}
                                            >
                                                {#if tx.categoryId}
                                                    <CategoryBadge
                                                        name={$categories.find((category) => category.id === tx.categoryId)?.name || "Категория"}
                                                        color={$categories.find((category) => category.id === tx.categoryId)?.color || ""}
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
                                                {#each $categories as category}
                                                    <li>
                                                        <button type="button" on:click={(event) => handleCategorySelect(event, tx.id, category.id)}>
                                                            <CategoryBadge name={category.name} color={category.color} primaryWhenNoColor={true} />
                                                        </button>
                                                    </li>
                                                {/each}
                                            </ul>
                                        </div>
                                    </td>
                                    <td class="text-right font-medium">{formatSignedCents(tx.amount, tx.entryType)}</td>
                                    <td>{tx.currency}</td>
                                </tr>
                            {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            {/if}
        </div>
    </div>
</section>

<style>
</style>
