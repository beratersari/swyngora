/**
 * API base URL for RTK Query.
 *
 * - Empty / unset → **same origin** (recommended in dev). Vite proxies `/api` and
 *   `/health` to the Go backend. This works when you open the UI via WSL IP
 *   (e.g. http://172.x.x.x:5174) from Windows, where `localhost:8080` would
 *   incorrectly point at Windows instead of the Linux backend.
 * - Set `VITE_API_BASE_URL=http://127.0.0.1:8080` only when the browser and API
 *   share the same host (native Linux browser, or Windows with port-forward).
 */
function resolveApiBaseUrl(): string {
  const raw = import.meta.env.VITE_API_BASE_URL;
  if (raw === undefined || raw === null) {
    return '';
  }
  const trimmed = String(raw).trim();
  if (trimmed === '' || trimmed === '/') {
    return '';
  }
  return trimmed.replace(/\/+$/, '');
}

function resolveClientId(): string {
  const raw = import.meta.env.VITE_CLIENT_ID;
  if (raw === undefined || raw === null) return '';
  return String(raw).trim();
}

export const env = {
  /** Empty string = relative same-origin requests (use Vite proxy in dev). */
  apiBaseUrl: resolveApiBaseUrl(),
  /** Human-readable label for UI */
  apiBaseUrlLabel: (() => {
    const base = resolveApiBaseUrl();
    return base === '' ? '(same origin / Vite proxy)' : base;
  })(),
  /**
   * Optional fixed client id for watchlist (`X-Client-Id`).
   * When empty, runtime generates/persists one via getOrCreateClientId().
   */
  clientId: resolveClientId(),
} as const;
