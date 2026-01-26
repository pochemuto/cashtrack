<script lang="ts">
    import {onMount} from "svelte";
    import {resolveApiUrl} from "$lib/url";
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
    };

    type TransactionListResponse = {
        items: TransactionItem[];
        summary: TransactionSummary;
    };

    type CategoryItem = {
        id: number;
        name: string;
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

    function formatSummaryAmount(value: string): string {
        const parsed = Number(value);
        if (!Number.isFinite(parsed)) {
            return value;
        }
        return parsed.toFixed(2);
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

    function resetFilters() {
        fromDate = "";
        toDate = "";
        sourceFileId = "";
        entryType = "";
        searchText = "";
        categoryFilter = "";
        accountNumber = "";
        cardNumber = "";
        void loadTransactions();
    }

    onMount(() => {
        if ($user?.id) {
            lastUserId = $user.id;
            void loadTransactions();
            void loadCategories();
        }
    });

    $: if ($user?.id && $user.id !== lastUserId) {
        lastUserId = $user.id;
        void loadTransactions();
        void loadCategories();
    }

    $: if (fromDate && toDate) {
        const next = `${fromDate}/${toDate}`;
        if (calendarRange !== next) {
            calendarRange = next;
        }
    } else if (!fromDate && !toDate && calendarRange) {
        calendarRange = "";
    }

    $: if (calendarElement && calendarElement.value !== calendarRange) {
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
</script>

<svelte:head>
    <title>Transactions</title>
</svelte:head>

<section class="mx-auto w-full max-w-none">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Транзакции</h1>
                <p class="text-sm opacity-70">
                    Фильтруйте транзакции по датам, источнику и описанию.
                </p>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
                <div class="form-control lg:col-span-2">
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
                    <label class="label" for="entry-type">
                        <span class="label-text">Тип списания</span>
                    </label>
                    <select class="select select-bordered" id="entry-type" bind:value={entryType}>
                        <option value="">Все</option>
                        <option value="debit">Debit</option>
                        <option value="credit">Credit</option>
                    </select>
                </div>
                <div class="form-control">
                    <label class="label" for="search-text">
                        <span class="label-text">Поиск по описанию</span>
                    </label>
                    <input class="input input-bordered" type="text" id="search-text" bind:value={searchText} placeholder="например Uber" />
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

            <div class="flex flex-wrap gap-3">
                <button class="btn btn-primary" type="button" on:click={loadTransactions} disabled={loading}>
                    {loading ? "Загрузка..." : "Применить"}
                </button>
                <button class="btn btn-ghost" type="button" on:click={resetFilters} disabled={loading}>
                    Сбросить
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
                    </div>
                {/if}
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
                                    <select
                                        class="select select-bordered select-sm w-full max-w-md min-w-[220px]"
                                        value={tx.category_id ?? ""}
                                        disabled={categoriesLoading || !categories.length || categoryUpdates[tx.id]}
                                        on:change={(event) => {
                                            const value = (event.target as HTMLSelectElement).value;
                                            const categoryId = value ? Number(value) : null;
                                            void updateTransactionCategory(tx.id, categoryId);
                                        }}
                                    >
                                        <option value="" disabled>
                                            Без категории
                                        </option>
                                        {#each categories as category}
                                            <option value={category.id}>{category.name}</option>
                                        {/each}
                                    </select>
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
