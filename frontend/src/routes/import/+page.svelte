<script lang="ts">
    import {onMount} from "svelte";
    import {resolveApiUrl} from "$lib/url";
    import {user} from "../../user";

    type UploadStatus = "idle" | "uploading" | "success" | "error";

    let file: File | null = null;
    let status: UploadStatus = "idle";
    let errorMessage = "";
    let uploadedName = "";
    let uploadedSize = 0;
    let reports: ReportItem[] = [];
    let listError = "";
    let loadingReports = false;
    let loadedForUserId: number | null = null;
    let deletingReportId: number | null = null;
    let toastMessage = "";
    let toastTimeout: ReturnType<typeof setTimeout> | null = null;

    type ReportItem = {
        id: number;
        filename: string;
        size_bytes: number;
        status: string;
        uploaded_at: string;
        status_description?: string | null;
    };

    function formatBytes(size: number): string {
        if (size < 1024) {
            return `${size} B`;
        }
        const kb = size / 1024;
        if (kb < 1024) {
            return `${kb.toFixed(1)} KB`;
        }
        const mb = kb / 1024;
        return `${mb.toFixed(1)} MB`;
    }

    function handleFileChange(event: Event) {
        const target = event.currentTarget as HTMLInputElement;
        file = target.files?.[0] ?? null;
        status = "idle";
        errorMessage = "";
    }

    async function handleUpload() {
        if (!file) {
            status = "error";
            errorMessage = "Выберите CSV файл.";
            return;
        }

        status = "uploading";
        errorMessage = "";
        const formData = new FormData();
        formData.append("file", file);

        try {
            const response = await fetch(resolveApiUrl("api/reports/upload"), {
                method: "POST",
                body: formData,
                credentials: "include",
            });

            if (response.ok) {
                uploadedName = file.name;
                uploadedSize = file.size;
                file = null;
                status = "success";
                showToast("Файл загружен.");
                await loadReports();
                return;
            }

            if (response.status === 401) {
                errorMessage = "Нужно войти в аккаунт, чтобы загрузить отчет.";
            } else {
                const text = (await response.text()).trim();
                errorMessage = text || "Не удалось загрузить файл.";
            }
            status = "error";
        } catch {
            status = "error";
            errorMessage = "Не удалось загрузить файл.";
        }
    }

    function formatDate(value: string): string {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        return date.toLocaleString("ru-RU", {
            year: "numeric",
            month: "short",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    async function loadReports() {
        if (!$user || !$user.id) {
            reports = [];
            listError = "";
            loadingReports = false;
            return;
        }
        loadingReports = true;
        listError = "";

        try {
            const response = await fetch(resolveApiUrl("api/reports"), {
                credentials: "include",
            });
            if (!response.ok) {
                if (response.status === 401) {
                    reports = [];
                    listError = "Нужен вход для просмотра списка отчетов.";
                    return;
                }
                listError = "Не удалось загрузить список отчетов.";
                return;
            }
            reports = (await response.json()) as ReportItem[];
        } catch {
            listError = "Не удалось загрузить список отчетов.";
        } finally {
            loadingReports = false;
        }
    }

    async function handleDeleteReport(report: ReportItem) {
        const confirmed = confirm(
            `Удалить файл "${report.filename}"? Все транзакции, загруженные из этого файла, будут удалены.`,
        );
        if (!confirmed) {
            return;
        }

        deletingReportId = report.id;
        listError = "";
        try {
            const response = await fetch(resolveApiUrl(`api/reports/delete?id=${report.id}`), {
                method: "DELETE",
                credentials: "include",
            });
            if (!response.ok) {
                if (response.status === 401) {
                    listError = "Нужен вход для удаления отчета.";
                    return;
                }
                listError = "Не удалось удалить отчет.";
                return;
            }
            await loadReports();
            showToast("Файл удален.");
        } catch {
            listError = "Не удалось удалить отчет.";
        } finally {
            deletingReportId = null;
        }
    }

    function showToast(message: string) {
        toastMessage = message;
        if (toastTimeout) {
            clearTimeout(toastTimeout);
        }
        toastTimeout = setTimeout(() => {
            toastMessage = "";
            toastTimeout = null;
        }, 3000);
    }

    onMount(() => {
        if ($user?.id) {
            loadedForUserId = $user.id;
            void loadReports();
        }
    });

    $: if ($user?.id && $user.id !== loadedForUserId) {
        loadedForUserId = $user.id;
        void loadReports();
    }

    $: if ($user === undefined) {
        reports = [];
        listError = "";
        loadingReports = false;
    }
</script>

<svelte:head>
    <title>Import Report</title>
</svelte:head>

<section class="mx-auto w-full max-w-2xl">
    {#if toastMessage}
        <div class="toast toast-top toast-end z-50">
            <div class="alert alert-success">
                <span>{toastMessage}</span>
            </div>
        </div>
    {/if}
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Импорт финансового отчета</h1>
                <p class="text-sm opacity-70">
                    Импортируйте CSV файл, он будет сохранен в базе данных без изменений.
                </p>
            </div>

            {#if $user === undefined}
                <div class="alert alert-warning">
                    <span>Войдите в аккаунт, чтобы загрузить отчет.</span>
                </div>
            {/if}

            <div class="form-control gap-3">
                <label class="label">
                    <span class="label-text">CSV файл</span>
                </label>
                <input
                    class="file-input file-input-bordered w-full"
                    type="file"
                    accept=".csv,text/csv"
                    on:change={handleFileChange}
                />
            </div>

            <div class="flex flex-wrap items-center gap-3">
                <button
                    class="btn btn-primary"
                    type="button"
                    on:click={handleUpload}
                    disabled={status === "uploading" || !file}
                >
                    {status === "uploading" ? "Импорт..." : "Импортировать"}
                </button>
                {#if file}
                    <span class="text-sm opacity-70">{file.name} · {formatBytes(file.size)}</span>
                {/if}
            </div>

            {#if status === "error"}
                <div class="alert alert-error">
                    <span>{errorMessage}</span>
                </div>
            {/if}

            <div class="divider"></div>

            <div class="space-y-3">
                <h2 class="text-lg font-semibold">Загруженные файлы</h2>
                {#if loadingReports}
                    <div class="text-sm opacity-70">Загрузка списка...</div>
                {:else if listError}
                    <div class="alert alert-error">
                        <span>{listError}</span>
                    </div>
                {:else if reports.length === 0}
                    <div class="text-sm opacity-70">Пока нет загруженных отчетов.</div>
                {:else}
                    <div class="overflow-x-auto">
                        <table class="table">
                            <thead>
                            <tr>
                                <th>Файл</th>
                                <th>Дата загрузки</th>
                                <th>Размер</th>
                                <th>Статус</th>
                                <th></th>
                            </tr>
                            </thead>
                            <tbody>
                            {#each reports as report}
                                <tr>
                                    <td>
                                        <a
                                            class="link link-primary"
                                            href={resolveApiUrl(`api/reports/download?id=${report.id}`)}
                                        >
                                            {report.filename}
                                        </a>
                                    </td>
                                    <td>{formatDate(report.uploaded_at)}</td>
                                    <td>{formatBytes(report.size_bytes)}</td>
                                    <td>
                                        {#if report.status_description}
                                            <span class="tooltip tooltip-left" data-tip={report.status_description}>
                                                <span
                                                    class:text-error={report.status === "failed"}
                                                    class:font-medium={report.status === "processed"}
                                                >
                                                    {report.status}
                                                </span>
                                            </span>
                                        {:else}
                                            <span class:font-medium={report.status === "processed"}>{report.status}</span>
                                        {/if}
                                    </td>
                                    <td class="text-right">
                                        <div class="dropdown dropdown-left">
                                            <button
                                                class="btn btn-ghost btn-xs"
                                                type="button"
                                                disabled={deletingReportId === report.id}
                                            >
                                                ⋮
                                            </button>
                                            <ul class="menu dropdown-content rounded-box bg-base-100 p-2 shadow">
                                                <li>
                                                    <button type="button" on:click={() => handleDeleteReport(report)}>
                                                        Удалить
                                                    </button>
                                                </li>
                                            </ul>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </div>
        </div>
    </div>
</section>
