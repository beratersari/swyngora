/** Candle shape from Swyngora API (OHLCV fields are strings). */
export type ApiCandle = {
  openTime: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: string;
  closeTime?: string;
};

/** Lightweight Charts candlestick bar (time as UTCTimestamp seconds). */
export type ChartCandle = {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
};

/** Map API candles to lightweight-charts candlestick data. */
export function apiCandlesToChart(candles: ApiCandle[]): ChartCandle[] {
  return candles
    .map((c) => {
      const openTimeMs = Date.parse(c.openTime);
      const open = Number(c.open);
      const high = Number(c.high);
      const low = Number(c.low);
      const close = Number(c.close);
      if (
        !Number.isFinite(openTimeMs) ||
        !Number.isFinite(open) ||
        !Number.isFinite(high) ||
        !Number.isFinite(low) ||
        !Number.isFinite(close)
      ) {
        return null;
      }
      return {
        time: Math.floor(openTimeMs / 1000),
        open,
        high,
        low,
        close,
      };
    })
    .filter((x): x is ChartCandle => x !== null);
}

/** Prefer the longer live series (full window over the first-paint slice). */
export function preferLongerCandleSeries(
  a: readonly ApiCandle[] | undefined,
  b: readonly ApiCandle[] | undefined,
): readonly ApiCandle[] | undefined {
  const al = a?.length ?? 0;
  const bl = b?.length ?? 0;
  if (bl >= al && bl > 0) return b;
  if (al > 0) return a;
  return undefined;
}

const CARRY_FORWARD_MAX_BARS = 200;

function visibleLastPriceWick(last: ApiCandle): { high: string; low: string } {
  const close = Number(last.close);
  const high = Number(last.high);
  const low = Number(last.low);
  if (Number.isFinite(high) && Number.isFinite(low) && high > low) {
    return { high: last.high, low: last.low };
  }
  const n = Number.isFinite(close) ? close : 0;
  const tick = n === 0 ? 1e-8 : Math.abs(n) * 0.002;
  return { high: String(n + tick), low: String(n - tick) };
}

/** Seconds per bar. `1M` is ~30d (intervalToSeconds treats it as 0). */
export function candleIntervalSeconds(interval: string): number {
  const raw = (interval ?? '').trim();
  if (raw === '1M') return 30 * 86400;
  const m = /^(\d+)([mhdw])$/.exec(raw);
  if (!m) return 0;
  const n = Number(m[1]);
  if (!Number.isFinite(n) || n <= 0) return 0;
  switch (m[2]) {
    case 'm':
      return n * 60;
    case 'h':
      return n * 3600;
    case 'd':
      return n * 86400;
    case 'w':
      return n * 604800;
    default:
      return 0;
  }
}

/**
 * After the last venue print, keep drawing candles through `untilMs` at the
 * last close. High/low reuse the last real wick so Lightweight Charts still
 * paints a body (O=H=L=C bars are invisible).
 */
export function carryForwardCandlesUntil(
  candles: readonly ApiCandle[],
  untilMs: number,
  intervalSec: number,
): ApiCandle[] {
  if (!candles.length || intervalSec <= 0 || !Number.isFinite(untilMs)) {
    return [...candles];
  }
  const last = candles[candles.length - 1]!;
  const lastOpen = Date.parse(last.openTime);
  if (!Number.isFinite(lastOpen)) return [...candles];
  const step = intervalSec * 1000;
  const px = last.close;
  const { high, low } = visibleLastPriceWick(last);
  const out = [...candles];
  let t = lastOpen + step;
  let n = 0;
  while (t < untilMs && n < CARRY_FORWARD_MAX_BARS) {
    out.push({
      openTime: new Date(t).toISOString(),
      open: px,
      high,
      low,
      close: px,
      volume: '0',
      closeTime: new Date(t + step - 1).toISOString(),
    });
    t += step;
    n += 1;
  }
  return out;
}

/** Keep candles with complete OHLC + openTime. */
export function filterValidApiCandles(
  candles: readonly (Partial<ApiCandle> | undefined)[] | undefined,
): ApiCandle[] {
  if (!candles?.length) return [];
  const out: ApiCandle[] = [];
  for (const c of candles) {
    if (!c?.openTime || !c.open || !c.high || !c.low || !c.close) continue;
    if (!Number.isFinite(Date.parse(c.openTime))) continue;
    out.push({
      openTime: c.openTime,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
      volume: c.volume ?? '0',
      closeTime: c.closeTime,
    });
  }
  return out;
}

/**
 * Merge candle series by openTime (later list wins on conflict), sorted ascending.
 * Used to stitch live polls with older history pages.
 */
export function mergeCandleHistory(
  older: readonly ApiCandle[],
  newer: readonly ApiCandle[],
): ApiCandle[] {
  const byTime = new Map<string, ApiCandle>();
  for (const c of older) {
    if (c.openTime) byTime.set(c.openTime, c);
  }
  for (const c of newer) {
    if (c.openTime) byTime.set(c.openTime, c);
  }
  return [...byTime.values()].sort(
    (a, b) => Date.parse(a.openTime) - Date.parse(b.openTime),
  );
}

/** openTime ms of the oldest bar, or null if empty. */
export function oldestCandleOpenTimeMs(candles: readonly ApiCandle[]): number | null {
  if (!candles.length) return null;
  let min = Number.POSITIVE_INFINITY;
  for (const c of candles) {
    const ms = Date.parse(c.openTime);
    if (Number.isFinite(ms) && ms < min) min = ms;
  }
  return Number.isFinite(min) ? min : null;
}

/**
 * Trim from the oldest side so length ≤ maxBars (keeps the newest bars).
 */
export function trimCandlesToMax(candles: readonly ApiCandle[], maxBars: number): ApiCandle[] {
  if (maxBars <= 0 || candles.length <= maxBars) return [...candles];
  return candles.slice(candles.length - maxBars);
}