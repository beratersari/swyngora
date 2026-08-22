import { i18n } from '@/libs/i18n';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function getRtkErrorCode(error: unknown): string | undefined {
  if (!isRecord(error) || !('data' in error) || !isRecord(error.data)) return undefined;
  const data = error.data;
  if (typeof data.code === 'string' && data.code.trim()) return data.code.trim();
  if (isRecord(data.error) && typeof data.error.code === 'string' && data.error.code.trim()) {
    return data.error.code.trim();
  }
  return undefined;
}

export function getRtkErrorStatus(error: unknown): number | undefined {
  if (isRecord(error) && typeof error.status === 'number') return error.status;
  return undefined;
}

export function rtkErrorMessage(
  error: unknown,
  options?: {
    resource?: string;
    codeMessages?: Partial<Record<string, string>>;
    statusMessages?: Partial<Record<number, string>>;
  },
): string {
  const t = i18n.t.bind(i18n);
  if (!error) {
    return options?.resource
      ? t('common:errors.failedToLoad', { resource: options.resource })
      : t('common:errors.requestFailed');
  }

  const code = getRtkErrorCode(error);
  if (code && options?.codeMessages?.[code]) {
    return options.codeMessages[code]!;
  }
  const status = getRtkErrorStatus(error);
  if (status != null && options?.statusMessages?.[status]) {
    return options.statusMessages[status]!;
  }

  if (isRecord(error)) {
    if ('status' in error && error.status === 'FETCH_ERROR') {
      return t('common:errors.network');
    }
    if ('data' in error && isRecord(error.data)) {
      const data = error.data;
      if (typeof data.message === 'string' && data.message) return data.message;
      if (isRecord(data.error) && typeof data.error.message === 'string') {
        return data.error.message;
      }
    }
    if ('error' in error && typeof error.error === 'string' && error.error) {
      return error.error;
    }
    if ('message' in error && typeof error.message === 'string' && error.message) {
      return error.message;
    }
    if ('status' in error && typeof error.status === 'number') {
      return t('common:errors.requestFailedStatus', { status: error.status });
    }
  }

  return options?.resource
    ? t('common:errors.failedToLoad', { resource: options.resource })
    : t('common:errors.requestFailed');
}
