export function formatCents(value: bigint): string {
    const sign = value < 0n ? "-" : "";
    const absValue = value < 0n ? -value : value;
    const whole = absValue / 100n;
    const fraction = absValue % 100n;
    return `${sign}${whole.toString()}.${fraction.toString().padStart(2, "0")}`;
}

export function formatSignedCents(value: bigint, entryType?: string): string {
    const formatted = formatCents(value);
    if (entryType === "credit" && value > 0n) {
        return `+${formatted}`;
    }
    return formatted;
}

export function centsToNumber(value: bigint): number {
    return Number(value) / 100;
}

function formatAmount(value: number, clampNegative: boolean): string {
    const safeValue = Number.isFinite(value) ? value : 0;
    const normalized = clampNegative ? Math.max(safeValue, 0) : safeValue;
    const sign = !clampNegative && normalized < 0 ? "-" : "";
    const absValue = Math.abs(normalized);
    if (absValue >= 100) {
        const grouped = Math.round(absValue)
            .toString()
            .replace(/\B(?=(\d{3})+(?!\d))/g, " ");
        return `${sign}${grouped}`;
    }
    const [integerPart, fractionalPart] = absValue.toFixed(2).split(".");
    const grouped = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, " ");
    return `${sign}${grouped}.${fractionalPart}`;
}

export function formatCurrencyAmount(value: number, currency: string, clampNegative = false): string {
    const formatted = formatAmount(value, clampNegative);
    return currency ? `${formatted} ${currency}` : formatted;
}

export function formatChfAmount(value: number): string {
    return formatCurrencyAmount(value, "CHF", true);
}
