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
    };

    let transactions: TransactionItem[] = [];
    let loading = false;
    let listError = "";

    let fromDate = "";
    let toDate = "";
    let sourceFileId = "";
    let entryType = "";
    let searchText = "";
    let accountNumber = "";
    let cardNumber = "";

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

    async function loadTransactions() {
        if (!$user || !$user.id) {
            transactions = [];
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
                    return;
                }
                listError = "Не удалось загрузить транзакции.";
                return;
            }
            transactions = (await response.json()) as TransactionItem[];
        } catch {
            listError = "Не удалось загрузить транзакции.";
        } finally {
            loading = false;
        }
    }

    function resetFilters() {
        fromDate = "";
        toDate = "";
        sourceFileId = "";
        entryType = "";
        searchText = "";
        accountNumber = "";
        cardNumber = "";
        void loadTransactions();
    }

    onMount(() => {
        if ($user?.id) {
            void loadTransactions();
        }
    });
</script>

<svelte:head>
    <title>Transactions</title>
</svelte:head>

<section class="mx-auto w-full max-w-6xl">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Транзакции</h1>
                <p class="text-sm opacity-70">
                    Фильтруйте транзакции по датам, источнику и описанию.
                </p>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Дата с</span>
                    </label>
                    <input class="input input-bordered" type="date" bind:value={fromDate} />
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Дата по</span>
                    </label>
                    <input class="input input-bordered" type="date" bind:value={toDate} />
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Источник (ID файла)</span>
                    </label>
                    <input class="input input-bordered" type="text" bind:value={sourceFileId} placeholder="например 12" />
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Тип списания</span>
                    </label>
                    <select class="select select-bordered" bind:value={entryType}>
                        <option value="">Все</option>
                        <option value="debit">Debit</option>
                        <option value="credit">Credit</option>
                    </select>
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Поиск по описанию</span>
                    </label>
                    <input class="input input-bordered" type="text" bind:value={searchText} placeholder="например Uber" />
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Номер счета</span>
                    </label>
                    <input class="input input-bordered" type="text" bind:value={accountNumber} />
                </div>
                <div class="form-control">
                    <label class="label">
                        <span class="label-text">Номер карты</span>
                    </label>
                    <input class="input input-bordered" type="text" bind:value={cardNumber} />
                </div>
            </div>

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

            {#if loading}
                <div class="text-sm opacity-70">Загрузка транзакций...</div>
            {:else if transactions.length === 0}
                <div class="text-sm opacity-70">Нет транзакций по выбранным фильтрам.</div>
            {:else}
                <div class="overflow-x-auto">
                    <table class="table">
                        <thead>
                        <tr>
                            <th>Дата</th>
                            <th>Описание</th>
                            <th class="text-right">Сумма</th>
                            <th>Валюта</th>
                            <th>Источник</th>
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
                                <td class="text-right font-medium">{amountWithSign(tx.amount, tx.entry_type)}</td>
                                <td>{tx.currency}</td>
                                <td class="text-xs">
                                    {#if tx.source_account_number}
                                        <div>Счет: {tx.source_account_number}</div>
                                    {/if}
                                    {#if tx.source_card_number}
                                        <div>Карта: {tx.source_card_number}</div>
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    </div>
</section>
