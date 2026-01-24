const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/";
const ABSOLUTE_URL_RE = /^https?:\/\//i;

export function getApiBaseUrl(): string {
    return API_BASE_URL;
}

export function resolveApiUrl(path: string): string {
    if (ABSOLUTE_URL_RE.test(path)) {
        return path;
    }

    if (ABSOLUTE_URL_RE.test(API_BASE_URL)) {
        return new URL(path, API_BASE_URL).toString();
    }

    const base = API_BASE_URL === "/" ? "" : API_BASE_URL.replace(/\/$/, "");
    const normalizedPath = path.startsWith("/") ? path : `/${path}`;
    const resolved = `${base}${normalizedPath}`;
    return resolved === "" ? "/" : resolved;
}
