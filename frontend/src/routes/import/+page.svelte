<script lang="ts">
    import {onMount} from "svelte";
    import {Reports} from "$lib/api";
    import type {ReportInfo} from "$lib/gen/api/v1/reports_pb";
    import {Code, ConnectError} from "@connectrpc/connect";
    import {user} from "../../user";

    type UploadStatus = "idle" | "uploading" | "success" | "error";

    let file: File | null = null;
    let status: UploadStatus = "idle";
    let errorMessage = "";
    let uploadedName = "";
    let uploadedSize = 0;
    let reports: ReportInfo[] = [];
    let listError = "";
    let loadingReports = false;
    let loadedForUserId: number | null = null;
    let deletingReportId: number | null = null;
    let toastMessage = "";
    let toastTimeout: ReturnType<typeof setTimeout> | null = null;

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
        try {
            const buffer = await file.arrayBuffer();
            await Reports.uploadReport({
                filename: file.name,
                data: new Uint8Array(buffer),
                contentType: file.type || "application/octet-stream",
            });
            uploadedName = file.name;
            uploadedSize = file.size;
            file = null;
            status = "success";
            showToast("Файл загружен.");
            await loadReports();
            return;
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                errorMessage = "Нужно войти в аккаунт, чтобы загрузить отчет.";
            } else {
                errorMessage = "Не удалось загрузить файл.";
            }
            status = "error";
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
            const response = await Reports.listReports({});
            reports = response.reports ?? [];
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                reports = [];
                listError = "Нужен вход для просмотра списка отчетов.";
                return;
            }
            listError = "Не удалось загрузить список отчетов.";
        } finally {
            loadingReports = false;
        }
    }

    async function handleDeleteReport(report: ReportInfo) {
        const confirmed = confirm(
            `Удалить файл "${report.filename}"? Все транзакции, загруженные из этого файла, будут удалены.`,
        );
        if (!confirmed) {
            return;
        }

        deletingReportId = report.id;
        listError = "";
        try {
            await Reports.deleteReport({id: report.id});
            await loadReports();
            showToast("Файл удален.");
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                listError = "Нужен вход для удаления отчета.";
                return;
            }
            listError = "Не удалось удалить отчет.";
        } finally {
            deletingReportId = null;
        }
    }

    async function handleDownloadReport(report: ReportInfo) {
        listError = "";
        try {
            const response = await Reports.downloadReport({id: report.id});
            const blob = new Blob([response.data as BlobPart], {
                type: response.contentType || "application/octet-stream",
            });
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.href = url;
            link.download = response.filename || report.filename;
            document.body.appendChild(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                listError = "Нужен вход для скачивания отчета.";
                return;
            }
            listError = "Не удалось скачать отчет.";
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
                                        <button
                                            class="link link-primary"
                                            type="button"
                                            on:click={() => handleDownloadReport(report)}
                                        >
                                            {report.filename}
                                        </button>
                                    </td>
                                    <td>{formatDate(report.uploadedAt)}</td>
                                    <td>{formatBytes(report.sizeBytes)}</td>
                                    <td>
                                        {#if report.statusDescription}
                                            <span class="tooltip tooltip-left" data-tip={report.statusDescription}>
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
