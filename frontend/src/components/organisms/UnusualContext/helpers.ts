import { formatCompactUsd } from '@/libs/utils/formatMarket';
import type { UnusualReadingView, UnusualTranslate } from './UnusualContext.types';

const HIDDEN_CURRENT = new Set(['no_sample', 'no_stored_history', 'no_book']);

/** Formats the current figure. A zero on an unknown reading is omitted. */
export function formatUnusualValue(reading: UnusualReadingView): string | null {
  const reason = reading.reason ?? '';
  const current = reading.current;
  if (HIDDEN_CURRENT.has(reason) && (current === undefined || current === 0)) {
    return null;
  }
  if (current === undefined || !Number.isFinite(current)) return null;
  switch (reading.unit) {
    case 'rate':
      return `${(current * 100).toFixed(4)}%`;
    case 'pct_change': {
      const sign = current > 0 ? '+' : '';
      return `${sign}${current.toFixed(2)}%`;
    }
    case 'notional':
      return `$${formatCompactUsd(current)}`;
    case 'signed_notional':
      if (current < 0) return `-$${formatCompactUsd(-current)}`;
      if (current > 0) return `+$${formatCompactUsd(current)}`;
      return '$0';
    case 'spread_pct':
      return `${current.toFixed(4)}%`;
    default:
      return String(current);
  }
}

/** Span label from the hours the stored samples actually cover. */
export function unusualSpan(spanHours: number | undefined, t: UnusualTranslate): string {
  const hours = spanHours ?? 0;
  if (hours < 1) return t('tape.unusual.span.underHour');
  if (hours < 36) {
    const count = Math.max(1, Math.round(hours));
    const key = count === 1 ? 'tape.unusual.span.hour' : 'tape.unusual.span.hours';
    return t(key, { count });
  }
  const count = Math.max(2, Math.round(hours / 24));
  return t('tape.unusual.span.days', { count });
}

export function unusualVenueName(exchange: string | undefined, t: UnusualTranslate): string {
  switch (exchange) {
    case 'binance':
      return t('tape.shareBinance');
    case 'bybit':
      return t('tape.shareBybit');
    case 'okx':
      return t('tape.shareOkx');
    default:
      return exchange ?? '';
  }
}

/** Localized sentence. Callers must not render `reading.summary`. */
export function unusualSentence(reading: UnusualReadingView, t: UnusualTranslate): string {
  if (reading.metric === 'print_size' || reading.reason === 'no_stored_history') {
    return t('tape.unusual.printSize');
  }
  if (reading.reason === 'no_book') return t('tape.unusual.noBook');
  const metric = t(`tape.unusual.metric.${reading.metric ?? 'funding'}`);
  if (reading.reason === 'no_sample') return t('tape.unusual.noSample', { metric });
  if (reading.band === 'unknown' || reading.reason === 'few_samples') {
    return t('tape.unusual.few', {
      metric,
      samples: reading.samples ?? 0,
      span: unusualSpan(reading.spanHours, t),
      need: reading.minSamples ?? 0,
    });
  }
  const kind =
    reading.label === 'predicted'
      ? t('tape.unusual.kind.predicted')
      : t('tape.unusual.kind.settled');
  return t(`tape.unusual.line.${reading.metric ?? 'funding'}`, {
    kind,
    current: formatUnusualValue(reading) ?? '',
    band: t(`tape.unusual.band.${reading.band ?? 'typical'}`),
    samples: reading.samples ?? 0,
    span: unusualSpan(reading.spanHours, t),
    percentile: Math.round(reading.percentile ?? 0),
  });
}
