/**
 * API base URL for RTK Query.
 *
 * Chrome (Vite web): empty → same origin; Vite proxies /api and /health to Go backend.
 * Native later: use absolute URL (Android emulator 10.0.2.2:8080).
 */
function resolveApiBaseUrl(): string {
  const raw = (import.meta as ImportMeta & { env?: Record<string, string> }).env?.VITE_API_BASE_URL;
  if (raw === undefined || raw === null) {
    return '';
  }
  const trimmed = String(raw).trim();
  if (trimmed === '' || trimmed === '/') {
    return '';
  }
  return trimmed.replace(/\/+$/, '');
}

export const env = {
  /** Empty string = relative same-origin requests (use Vite proxy in web dev). */
  apiBaseUrl: resolveApiBaseUrl(),
  apiBaseUrlLabel: (() => {
    const base = resolveApiBaseUrl();
    return base === '' ? '(same origin / Vite proxy)' : base;
  })(),
  /** Primary surface for this scaffold is Chrome via `npm run web`. */
  runtime: 'web' as const,
} as const;
