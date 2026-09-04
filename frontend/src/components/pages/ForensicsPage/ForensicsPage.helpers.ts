import {
  DEFAULT_FORENSICS_DURING,
  DEFAULT_FORENSICS_EXCHANGE,
  DEFAULT_FORENSICS_LOOKBACK,
  DEFAULT_FORENSICS_SYMBOL,
  DEFAULT_FORENSICS_VIEW,
  DEFAULT_FORENSICS_WINDOW,
  FORENSICS_DURINGS,
  FORENSICS_EXCHANGES,
  FORENSICS_LOOKBACKS,
  FORENSICS_VIEWS,
  FORENSICS_WINDOWS,
} from './ForensicsPage.constants';
import type { ForensicsDuring, ForensicsExchange, ForensicsWindow } from './ForensicsPage.types';
import type { ForensicsLookback, ForensicsView } from '@/libs/types/forensics.types';

export function parseForensicsSymbol(raw: string | null): string {
  const s = raw?.trim().toUpperCase() ?? '';
  return s || DEFAULT_FORENSICS_SYMBOL;
}

export function parseForensicsExchange(raw: string | null): ForensicsExchange {
  const s = (raw ?? '').trim().toLowerCase();
  return (FORENSICS_EXCHANGES as readonly string[]).includes(s)
    ? (s as ForensicsExchange)
    : DEFAULT_FORENSICS_EXCHANGE;
}

export function parseForensicsWindow(raw: string | null): ForensicsWindow {
  const s = (raw ?? '').trim();
  return (FORENSICS_WINDOWS as readonly string[]).includes(s)
    ? (s as ForensicsWindow)
    : DEFAULT_FORENSICS_WINDOW;
}

export function parseForensicsView(raw: string | null): ForensicsView {
  const s = (raw ?? '').trim().toLowerCase();
  return (FORENSICS_VIEWS as readonly string[]).includes(s)
    ? (s as ForensicsView)
    : DEFAULT_FORENSICS_VIEW;
}

export function parseForensicsLookback(raw: string | null): ForensicsLookback {
  const s = (raw ?? '').trim();
  return (FORENSICS_LOOKBACKS as readonly string[]).includes(s)
    ? (s as ForensicsLookback)
    : DEFAULT_FORENSICS_LOOKBACK;
}

export function parseForensicsDuring(raw: string | null): ForensicsDuring {
  const s = (raw ?? '').trim();
  return (FORENSICS_DURINGS as readonly string[]).includes(s)
    ? (s as ForensicsDuring)
    : DEFAULT_FORENSICS_DURING;
}

/** Default event time: 1 hour ago, rounded down to 15 minutes (UTC ISO). */
export function defaultForensicsAt(now = Date.now()): string {
  const d = new Date(now - 60 * 60 * 1000);
  d.setUTCMinutes(Math.floor(d.getUTCMinutes() / 15) * 15, 0, 0);
  return d.toISOString();
}

export function parseForensicsAt(raw: string | null, now = Date.now()): string {
  if (!raw?.trim()) return defaultForensicsAt(now);
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return defaultForensicsAt(now);
  return d.toISOString();
}

export function utcToDatetimeLocal(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function datetimeLocalToUtc(local: string): string {
  const d = new Date(local);
  if (Number.isNaN(d.getTime())) return '';
  return d.toISOString();
}


