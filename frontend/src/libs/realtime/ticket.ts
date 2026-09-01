import { env } from '@/config/env';
import { getBrowserApiToken } from '@/libs/utils/apiAuth';
import { getOrCreateClientId } from '@/libs/utils/clientId';

/** POST /api/v1/realtime/ticket — one-time, ~60s. Empty when the API is open or the mint fails. */
export async function mintRealtimeTicket(
  fetchImpl: typeof fetch = fetch,
  apiBaseUrl = env.apiBaseUrl,
  clientId = getOrCreateClientId(),
  authToken = getBrowserApiToken(),
): Promise<string> {
  const id = clientId.trim();
  if (!id) return '';
  const headers: Record<string, string> = { 'X-Client-Id': id, Accept: 'application/json' };
  const tok = authToken.trim();
  if (tok) headers.Authorization = `Bearer ${tok}`;
  const base = (apiBaseUrl || '').replace(/\/+$/, '');
  const url = `${base}/api/v1/realtime/ticket`;
  try {
    const res = await fetchImpl(url, { method: 'POST', headers });
    if (!res.ok) return '';
    const body = (await res.json()) as { ticket?: unknown };
    return typeof body.ticket === 'string' ? body.ticket.trim() : '';
  } catch {
    return '';
  }
}
