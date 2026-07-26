function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function rtkErrorMessage(
  error: unknown,
  options?: { resource?: string },
): string {
  if (!error) {
    return options?.resource
      ? `Failed to load ${options.resource}`
      : 'Request failed';
  }

  if (isRecord(error)) {
    if ('status' in error && error.status === 'FETCH_ERROR') {
      return 'Network error — is the backend running?';
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
      return `Request failed (${error.status})`;
    }
  }

  return options?.resource
    ? `Failed to load ${options.resource}`
    : 'Request failed';
}
