<script lang="ts">
    import {resolveApiUrl} from "$lib/url";
    import {user} from "../../user";

    type UploadStatus = "idle" | "uploading" | "success" | "error";

    let file: File | null = null;
    let status: UploadStatus = "idle";
    let errorMessage = "";
    let uploadedName = "";
    let uploadedSize = 0;

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
</script>

<svelte:head>
    <title>Upload Report</title>
</svelte:head>

<section class="mx-auto w-full max-w-2xl">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Загрузка финансового отчета</h1>
                <p class="text-sm opacity-70">
                    Загрузите CSV файл, он будет сохранен в базе данных без изменений.
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
                    {status === "uploading" ? "Загрузка..." : "Загрузить"}
                </button>
                {#if file}
                    <span class="text-sm opacity-70">{file.name} · {formatBytes(file.size)}</span>
                {/if}
            </div>

            {#if status === "success"}
                <div class="alert alert-success">
                    <span>Файл {uploadedName} ({formatBytes(uploadedSize)}) успешно сохранен.</span>
                </div>
            {:else if status === "error"}
                <div class="alert alert-error">
                    <span>{errorMessage}</span>
                </div>
            {/if}
        </div>
    </div>
</section>
