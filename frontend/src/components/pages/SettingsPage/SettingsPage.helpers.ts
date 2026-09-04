import { env } from '@/config/env';
import type { ImportJob } from '@/libs/api';
import { getOrCreateClientId } from '@/libs/utils';

export type CustomSignalImportPreview = {
  addNames: string[];
  updateNames: string[];
  willAdd: number;
  willUpdate: number;
};

/** Named custom-signal mixes from an import preview (new vs overwrite). */
export function customSignalImportPreview(job?: ImportJob | null): CustomSignalImportPreview {
  const sec = job?.sections?.customSignals;
  return {
    addNames: [...(sec?.addNames ?? [])],
    updateNames: [...(sec?.updateNames ?? [])],
    willAdd: sec?.willAdd ?? 0,
    willUpdate: sec?.willUpdate ?? 0,
  };
}

export function exportDownloadHref(downloadUrl?: string | null): string | null {
  if (!downloadUrl) return null;
  if (/^https?:\/\//i.test(downloadUrl)) return downloadUrl;
  const base = env.apiBaseUrl.replace(/\/+$/, '');
  const path = downloadUrl.startsWith('/') ? downloadUrl : `/${downloadUrl}`;
  return `${base}${path}`;
}

export function currentClientId(): string {
  return getOrCreateClientId();
}

/** Trade keys may become the browser session. Read keys must not — they 403 every mutation. */
export function createdKeySessionToken(
  secret?: string | null,
  permission?: string | null,
): string | null {
  const token = (secret ?? '').trim();
  if (!token) return null;
  if ((permission ?? '').trim().toLowerCase() !== 'trade') return null;
  return token;
}
