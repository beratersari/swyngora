import { i18n } from '@/libs/i18n';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function rtkErrorMessage(
  error: unknown,
  options?: { resource?: string },
): string {
  const t = i18n.t.bind(i18n);
  if (!error) {
    return options?.resource
      ? t('common:errors.failedToLoad', { resource: options.resource })
      : t('common:errors.requestFailed');
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
