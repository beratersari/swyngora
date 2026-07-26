import type { FetchBaseQueryError } from '@reduxjs/toolkit/query';
import type { SerializedError } from '@reduxjs/toolkit';

export type RtkErrorMessageOptions = {
  /**
   * Human label for the failed resource (e.g. "markets", "watchlist").
   * Used only in generic fallbacks — not required.
   */
  resource?: string;
  /**
   * Override default copy for specific HTTP/network statuses.
   * Keys may be numbers (HTTP) or strings (e.g. "FETCH_ERROR").
   * Call-site overrides always win when present.
   */
  statusMessages?: Partial<Record<number | string, string>>;
};

type ApiErrorBody = {
  error?: { code?: string; message?: string };
  message?: string;
};

const DEFAULT_STATUS_MESSAGES: Record<number | string, string> = {
  400: 'Invalid request. Check filters and try again.',
  401: 'You are not authorized to perform this action.',
  403: 'Access denied.',
  404: 'The requested resource was not found.',
  429: 'Rate limited by the API. Slow down and retry in a moment.',
  500: 'The server hit an unexpected error. Try again shortly.',
  502: 'Upstream data is unavailable right now. Retry shortly.',
  503: 'Service temporarily unavailable. Retry shortly.',
  FETCH_ERROR: 'Could not reach the API. Is the backend running?',
  PARSING_ERROR: 'Received an unexpected response from the API.',
  TIMEOUT_ERROR: 'The request timed out. Try again.',
  CUSTOM_ERROR: 'Request failed.',
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

/** Extract HTTP or RTK status from a query/mutation error. */
export function getRtkErrorStatus(error: unknown): number | string | undefined {
  if (!isRecord(error)) return undefined;
  if ('status' in error) {
    const status = error.status;
    if (typeof status === 'number' || typeof status === 'string') return status;
  }
  return undefined;
}

/** Best-effort message string from API JSON or SerializedError. */
export function getRtkErrorRawMessage(error: unknown): string | undefined {
  if (!isRecord(error)) return undefined;

  if ('data' in error) {
    const data = error.data;
    if (typeof data === 'string' && data.trim()) return data.trim();
    if (isRecord(data)) {
      const body = data as ApiErrorBody;
      const nested = body.error?.message;
      if (typeof nested === 'string' && nested.trim()) return nested.trim();
      if (typeof body.message === 'string' && body.message.trim()) return body.message.trim();
    }
  }

  if (typeof error.error === 'string' && error.error.trim() && error.status !== 'FETCH_ERROR') {
    // FETCH_ERROR's `error` field is often a low-level TypeError string — prefer status copy
    return error.error.trim();
  }
  if (typeof error.message === 'string' && error.message.trim()) return error.message.trim();

  return undefined;
}

/**
 * Map any RTK Query / mutation `error` value to a user-facing string.
 *
 * Priority:
 * 1. Call-site `statusMessages` override
 * 2. API / SerializedError message body
 * 3. Built-in status defaults (429, 502, FETCH_ERROR, …)
 * 4. Generic fallback (optionally with `resource`)
 */
export function rtkErrorMessage(error: unknown, options: RtkErrorMessageOptions = {}): string {
  const resource = options.resource?.trim();
  const status = getRtkErrorStatus(error);

  if (status !== undefined && options.statusMessages?.[status]) {
    return options.statusMessages[status]!;
  }

  const raw = getRtkErrorRawMessage(error);
  if (raw) return raw;

  if (status !== undefined && DEFAULT_STATUS_MESSAGES[status]) {
    return DEFAULT_STATUS_MESSAGES[status]!;
  }

  if (status !== undefined) {
    return resource
      ? `Failed to load ${resource} (${String(status)}).`
      : `Request failed (${String(status)}).`;
  }

  return resource
    ? `Could not load ${resource}. Please try again.`
    : 'Something went wrong. Please try again.';
}

/** Type guard for RTK fetch errors (optional use in callers). */
export function isFetchBaseQueryError(error: unknown): error is FetchBaseQueryError {
  return isRecord(error) && 'status' in error;
}

export function isSerializedError(error: unknown): error is SerializedError {
  return isRecord(error) && 'message' in error && !('status' in error);
}
