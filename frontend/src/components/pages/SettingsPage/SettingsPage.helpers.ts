import { env } from '@/config/env';
import { getOrCreateClientId } from '@/libs/utils';

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
