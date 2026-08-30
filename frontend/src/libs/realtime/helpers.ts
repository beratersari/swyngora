import { env } from '@/config/env';
import { getOrCreateClientId } from '@/libs/utils/clientId';
import { REALTIME_PATH, REALTIME_RECONNECT_MAX_MS, REALTIME_RECONNECT_MIN_MS } from './constants';
import type { RealtimeMessage, RealtimeSymbolRef } from './realtime.types';

/** Stable key for an exchange+symbol pair. */
export function symbolKey(ref: RealtimeSymbolRef): string {
  const ex = (ref.exchange || 'binance').trim().toLowerCase();
  const sym = (ref.symbol || '').trim().toUpperCase();
  return `${ex}:${sym}`;
}

export function normalizeSymbolRef(ref: RealtimeSymbolRef): RealtimeSymbolRef | null {
  const exchange = (ref.exchange || '').trim().toLowerCase();
  const symbol = (ref.symbol || '').trim().toUpperCase();
  if (!exchange || !symbol) return null;
  return { exchange, symbol };
}

export function uniqueSymbolRefs(refs: RealtimeSymbolRef[]): RealtimeSymbolRef[] {
  const seen = new Set<string>();
  const out: RealtimeSymbolRef[] = [];
  for (const raw of refs) {
    const n = normalizeSymbolRef(raw);
    if (!n) continue;
    const k = symbolKey(n);
    if (seen.has(k)) continue;
    seen.add(k);
    out.push(n);
  }
  return out;
}

/** Build the WebSocket URL (same-origin or VITE_API_BASE_URL). */
export function realtimeWsUrl(
  apiBaseUrl = env.apiBaseUrl,
  clientId = getOrCreateClientId(),
  locationHost?: string,
  locationProtocol?: string,
  ticket = '',
): string {
  const params = new URLSearchParams();
  const id = clientId.trim();
  if (id) params.set('clientId', id);
  const t = ticket.trim();
  if (t) params.set('ticket', t);
  const q = params.toString() ? `?${params.toString()}` : '';
  const base = (apiBaseUrl || '').trim();
  if (base) {
    const wsBase = base.replace(/^http:/i, 'ws:').replace(/^https:/i, 'wss:');
    return `${wsBase.replace(/\/+$/, '')}${REALTIME_PATH}${q}`;
  }
  const proto = locationProtocol === 'https:' ? 'wss:' : 'ws:';
  const host = locationHost || (typeof location !== 'undefined' ? location.host : 'localhost');
  return `${proto}//${host}${REALTIME_PATH}${q}`;
}

export function reconnectDelayMs(attempt: number): number {
  const exp = Math.min(REALTIME_RECONNECT_MAX_MS, REALTIME_RECONNECT_MIN_MS * 2 ** Math.max(0, attempt));
  return exp;
}

export function parseRealtimeMessage(raw: unknown): RealtimeMessage | null {
  if (!raw || typeof raw !== 'object') return null;
  const type = (raw as { type?: unknown }).type;
  if (typeof type !== 'string' || !type.trim()) return null;
  return raw as RealtimeMessage;
}
