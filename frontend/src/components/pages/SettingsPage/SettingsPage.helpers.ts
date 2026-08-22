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
