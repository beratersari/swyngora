import type { FetchBaseQueryError } from '@reduxjs/toolkit/query';
import type { SerializedError } from '@reduxjs/toolkit';
import { i18n } from '@/libs/i18n';

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
  /** Override copy for API error.code (e.g. catalog_unmapped). Wins over status. */
  codeMessages?: Partial<Record<string, string>>;
};

type ApiErrorBody = {
  /** Nested OpenAPI Error shape, or legacy flat string. */
  error?: { code?: string; message?: string } | string;
  message?: string;
  code?: string;
};

function defaultStatusMessage(status: number | string): string | undefined {
  const map: Record<number | string, string> = {
    400: i18n.t('common:errors.invalidRequest'),
    401: i18n.t('common:errors.unauthorized'),
    403: i18n.t('common:errors.forbidden'),
    404: i18n.t('common:errors.notFound'),
    429: i18n.t('common:errors.rateLimited'),
    500: i18n.t('common:errors.serverError'),
    502: i18n.t('common:errors.upstreamUnavailable'),
    503: i18n.t('common:errors.serviceUnavailable'),
    FETCH_ERROR: i18n.t('common:errors.network'),
    PARSING_ERROR: i18n.t('common:errors.parse'),
    TIMEOUT_ERROR: i18n.t('common:errors.timeout'),
    CUSTOM_ERROR: i18n.t('common:errors.requestFailed', { status: 'error' }),
  };
  return map[status];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

/** Extract OpenAPI `error.code` from a query/mutation error body. */
export function getRtkErrorCode(error: unknown): string | undefined {
  if (!isRecord(error) || !('data' in error) || !isRecord(error.data)) return undefined;
  const body = error.data as ApiErrorBody;
  if (typeof body.code === 'string' && body.code.trim()) return body.code.trim();
  if (isRecord(body.error) && typeof body.error.code === 'string' && body.error.code.trim()) {
    return body.error.code.trim();
  }
  return undefined;
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
      if (typeof body.error === 'string' && body.error.trim()) {
        // Prefer explicit message when present (legacy flat auth body).
        if (typeof body.message === 'string' && body.message.trim()) return body.message.trim();
        return body.error.trim();
      }
      if (isRecord(body.error)) {
        const nested = body.error.message;
        if (typeof nested === 'string' && nested.trim()) return nested.trim();
      }
      if (typeof body.message === 'string' && body.message.trim()) return body.message.trim();
    }
  }

  if (typeof error.error === 'string' && error.error.trim() && error.status !== 'FETCH_ERROR') {
    return error.error.trim();
  }
  if (typeof error.message === 'string' && error.message.trim()) return error.message.trim();

  return undefined;
}

/**
 * Map any RTK Query / mutation `error` value to a user-facing string.
 * Default status copy is localized via i18n.
 */
export function rtkErrorMessage(error: unknown, options: RtkErrorMessageOptions = {}): string {
  const resource = options.resource?.trim();
  const status = getRtkErrorStatus(error);
  const code = getRtkErrorCode(error);

  if (code && options.codeMessages?.[code]) {
    return options.codeMessages[code]!;
  }

  if (status !== undefined && options.statusMessages?.[status]) {
    return options.statusMessages[status]!;
  }

  const raw = getRtkErrorRawMessage(error);
  if (raw) return raw;

  if (status !== undefined) {
    const mapped = defaultStatusMessage(status);
    if (mapped) return mapped;
  }

  if (status !== undefined) {
    return resource
      ? i18n.t('common:errors.loadFailedStatus', { resource, status: String(status) })
      : i18n.t('common:errors.requestFailed', { status: String(status) });
  }

  return resource
    ? i18n.t('common:errors.loadFailed', { resource })
    : i18n.t('common:errors.generic');
}

/** Type guard for RTK fetch errors (optional use in callers). */
export function isFetchBaseQueryError(error: unknown): error is FetchBaseQueryError {
  return isRecord(error) && 'status' in error;
}

export function isSerializedError(error: unknown): error is SerializedError {
  return isRecord(error) && 'message' in error && !('status' in error);
}
