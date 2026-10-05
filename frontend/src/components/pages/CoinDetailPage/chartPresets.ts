/** One desk on the coin chart. All is every layer. A manual switch leaves the desk unmatched. */
export const CHART_PRESETS = ['all', 'positioning', 'orderFlow', 'liquidity'] as const;
export type ChartPreset = (typeof CHART_PRESETS)[number];

export type ChartDeskLayers = {
  showDelta: boolean;
  showCvd: boolean;
  showLiqLevels: boolean;
  showBook: boolean;
  showFootprint: boolean;
  showCrowd: boolean;
  showOi: boolean;
  showFunding: boolean;
  showLiqNotional: boolean;
  showLiqMap: boolean;
  showOpenLevels: boolean;
};

const LAYER_KEYS = [
  'showDelta',
  'showCvd',
  'showLiqLevels',
  'showBook',
  'showFootprint',
  'showCrowd',
  'showOi',
  'showFunding',
  'showLiqNotional',
  'showLiqMap',
  'showOpenLevels',
] as const satisfies readonly (keyof ChartDeskLayers)[];

const ALL_ON: ChartDeskLayers = {
  showDelta: true,
  showCvd: true,
  showLiqLevels: true,
  showBook: true,
  showFootprint: true,
  showCrowd: true,
  showOi: true,
  showFunding: true,
  showLiqNotional: true,
  showLiqMap: true,
  showOpenLevels: true,
};

const ALL_OFF: ChartDeskLayers = {
  showDelta: false,
  showCvd: false,
  showLiqLevels: false,
  showBook: false,
  showFootprint: false,
  showCrowd: false,
  showOi: false,
  showFunding: false,
  showLiqNotional: false,
  showLiqMap: false,
  showOpenLevels: false,
};

const PERP_VENUES = ['binance', 'bybit', 'okx'] as const;

export function isChartPreset(value: string): value is ChartPreset {
  return (CHART_PRESETS as readonly string[]).includes(value);
}

/** Layers that belong on one desk. Order flow keeps day and 4h CVD because those lines follow the CVD switch. */
export function chartPresetLayers(preset: ChartPreset): ChartDeskLayers {
  switch (preset) {
    case 'all':
      return { ...ALL_ON };
    case 'positioning':
      return {
        ...ALL_OFF,
        showOi: true,
        showFunding: true,
        showCrowd: true,
        showOpenLevels: true,
        showLiqLevels: true,
      };
    case 'orderFlow':
      return { ...ALL_OFF, showDelta: true, showCvd: true, showFootprint: true };
    case 'liquidity':
      return { ...ALL_OFF, showBook: true };
    default:
      return { ...ALL_ON };
  }
}

/** The desk whose layers match exactly, or null after a manual switch. */
export function matchingChartPreset(layers: ChartDeskLayers): ChartPreset | null {
  for (const preset of CHART_PRESETS) {
    const want = chartPresetLayers(preset);
    if (LAYER_KEYS.every((key) => want[key] === layers[key])) return preset;
  }
  return null;
}

type ReviewBucket = {
  leverage?: number;
  side?: string;
  horizon?: string;
  hitRate?: number;
  sizeScale?: number;
};

type ReviewVenue = { byLeverage?: ReviewBucket[] };

/** Later-print hit rate and kept size for this page's venue, on the strongest listed leverage. */
export function liquidationLevelReview(
  report:
    | {
        review?: { binance?: ReviewVenue; bybit?: ReviewVenue; okx?: ReviewVenue };
      }
    | null
    | undefined,
  venue: 'binance' | 'bybit' | 'okx',
  side: 'long' | 'short',
  leverages: number[],
): { leverage: number; hitRate: number; sizeScale: number } | null {
  const wanted = new Set(leverages.filter((n) => n > 0));
  if (wanted.size === 0) return null;
  const matches = (report?.review?.[venue]?.byLeverage ?? []).filter(
    (bucket) =>
      bucket.side === side &&
      bucket.leverage != null &&
      wanted.has(bucket.leverage) &&
      isFiniteNumber(bucket.hitRate) &&
      isFiniteNumber(bucket.sizeScale),
  );
  if (matches.length === 0) return null;
  const top = Math.max(...matches.map((bucket) => bucket.leverage ?? 0));
  const atTop = matches.filter((bucket) => bucket.leverage === top);
  const picked = atTop.find((bucket) => bucket.horizon === '1h') ?? atTop[0];
  if (!picked || picked.leverage == null || !isFiniteNumber(picked.hitRate) || !isFiniteNumber(picked.sizeScale)) {
    return null;
  }
  return { leverage: picked.leverage, hitRate: picked.hitRate, sizeScale: picked.sizeScale };
}

export type DeskSentence = {
  /** Backend sentence for the read that is actually drawn. */
  text: string;
  /** `combined` when every venue covered the window. */
  exchange: string;
  /** Venues on the report that did not cover the window. */
  withheld: string[];
};

type WindowedVenue = {
  exchange?: string;
  error?: string;
  summary?: string;
  complete?: boolean;
  windows?: { window?: string; summary?: string; complete?: boolean }[];
};

function pickWindow(venue: WindowedVenue | undefined): { summary?: string; complete?: boolean } | undefined {
  if (!venue) return undefined;
  return venue.windows?.find((row) => row.window === '15m') ?? venue.windows?.find((row) => row.window === '5m');
}

/**
 * Print size for the order-flow desk.
 * Combined is used only when that 15m (else 5m) window is complete.
 */
export function printSizeDeskNote(
  report: { venues?: WindowedVenue[]; combined?: WindowedVenue } | null | undefined,
  pageExchange: string,
): DeskSentence | null {
  return windowedDeskNote(report, pageExchange);
}

type AbsorptionVenue = {
  exchange?: string;
  error?: string;
  summary?: string;
  complete?: boolean;
};

/** Absorption for the order-flow desk. Combined only when that venue is complete. */
export function absorptionDeskNote(
  report: { venues?: AbsorptionVenue[]; combined?: AbsorptionVenue } | null | undefined,
  pageExchange: string,
): DeskSentence | null {
  if (!report) return null;
  const venues = report.venues ?? [];
  const withheld = venues
    .filter((venue) => venue.exchange && (venue.error || venue.complete === false || !venue.summary))
    .map((venue) => venue.exchange as string);
  const combined = report.combined;
  if (combined && combined.complete === true && !combined.error && combined.summary) {
    return { text: combined.summary, exchange: 'combined', withheld: [] };
  }
  const page = venues.find((venue) => venue.exchange === pageExchange && !venue.error && venue.complete !== false && venue.summary);
  if (page?.summary && page.exchange) {
    return {
      text: page.summary,
      exchange: page.exchange,
      withheld: withheld.filter((id) => id !== page.exchange),
    };
  }
  const any = venues.find((venue) => venue.exchange && !venue.error && venue.complete !== false && venue.summary);
  if (any?.summary && any.exchange) {
    return {
      text: any.summary,
      exchange: any.exchange,
      withheld: withheld.filter((id) => id !== any.exchange),
    };
  }
  return null;
}

function windowedDeskNote(
  report: { venues?: WindowedVenue[]; combined?: WindowedVenue } | null | undefined,
  pageExchange: string,
): DeskSentence | null {
  if (!report) return null;
  const venues = report.venues ?? [];
  const incomplete = (venue: WindowedVenue | undefined) => {
    const row = pickWindow(venue);
    return !venue?.exchange || Boolean(venue.error) || !row?.complete || !row.summary;
  };
  const withheld = venues.filter((venue) => incomplete(venue) && venue.exchange).map((venue) => venue.exchange as string);
  const combinedRow = pickWindow(report.combined);
  if (combinedRow?.complete && combinedRow.summary) {
    return { text: combinedRow.summary, exchange: 'combined', withheld: [] };
  }
  const page = venues.find((venue) => venue.exchange === pageExchange);
  const pageRow = pickWindow(page);
  if (page?.exchange && !page.error && pageRow?.complete && pageRow.summary) {
    return {
      text: pageRow.summary,
      exchange: page.exchange,
      withheld: withheld.filter((id) => id !== page.exchange),
    };
  }
  const any = venues.find((venue) => {
    const row = pickWindow(venue);
    return Boolean(venue.exchange) && !venue.error && row?.complete && row.summary;
  });
  const anyRow = pickWindow(any);
  if (any?.exchange && anyRow?.summary) {
    return {
      text: anyRow.summary,
      exchange: any.exchange,
      withheld: withheld.filter((id) => id !== any.exchange),
    };
  }
  return null;
}

type PerpSide = { notional?: string };
type PerpHeadline = { rangePct?: string; below?: PerpSide; above?: PerpSide; complete?: boolean };
type PerpVenue = { exchange?: string; error?: string; headline?: PerpHeadline };

export type PerpBidShareNote = {
  /** True only when Binance, Bybit, and OKX each cover the band and the combined headline is complete. */
  definite: boolean;
  bidPct: number | null;
  rangePct: string;
  missing: string[];
  recorded: { exchange: string; bidPct: number }[];
};

/** Bid share of this coin's perp book inside the headline band. A missing venue is not zero. */
export function perpBidShareNote(
  report: { rangePct?: string; venues?: PerpVenue[]; combined?: PerpVenue } | null | undefined,
): PerpBidShareNote | null {
  if (!report) return null;
  const venues = report.venues ?? [];
  const recorded: { exchange: string; bidPct: number }[] = [];
  const fresh = new Set<string>();
  for (const venue of venues) {
    if (!venue.exchange || venue.error) continue;
    const pct = bidPercent(venue.headline);
    if (pct == null || venue.headline?.complete === false) continue;
    fresh.add(venue.exchange);
    recorded.push({ exchange: venue.exchange, bidPct: pct });
  }
  const missing = PERP_VENUES.filter((id) => !fresh.has(id));
  const rangePct = report.combined?.headline?.rangePct || report.rangePct || '1';
  if (missing.length > 0) {
    return { definite: false, bidPct: null, rangePct, missing: [...missing], recorded };
  }
  const combinedPct = bidPercent(report.combined?.headline);
  if (combinedPct == null || report.combined?.headline?.complete === false || report.combined?.error) {
    return { definite: false, bidPct: null, rangePct, missing: [], recorded };
  }
  return { definite: true, bidPct: combinedPct, rangePct, missing: [], recorded };
}

export type ChartWallNote = {
  side: 'bid' | 'ask';
  price: string;
  behavior: string;
};

/** Largest live walls on this venue's book, up to three. */
export function chartWallNotes(
  walls:
    | {
        side?: string;
        price?: string;
        notional?: string;
        share?: number;
        behavior?: string;
      }[]
    | null
    | undefined,
): ChartWallNote[] {
  const rows = (walls ?? [])
    .map((wall) => {
      const notional = parseQty(wall.notional) ?? 0;
      const share = wall.share != null && Number.isFinite(wall.share) ? wall.share : 0;
      return { wall, notional, share };
    })
    .filter((row) => (row.wall.side === 'bid' || row.wall.side === 'ask') && row.wall.price && (row.notional > 0 || row.share > 0));
  rows.sort((a, b) => b.notional - a.notional || b.share - a.share);
  return rows.slice(0, 3).map((row) => ({
    side: row.wall.side as 'bid' | 'ask',
    price: row.wall.price as string,
    behavior: row.wall.behavior ?? '',
  }));
}

export function formatDeskPct(value: number): string {
  const rounded = Math.round(value * 10) / 10;
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1);
}

function bidPercent(headline: PerpHeadline | undefined): number | null {
  const below = parseQty(headline?.below?.notional);
  const above = parseQty(headline?.above?.notional);
  if (below == null || above == null || below + above <= 0) return null;
  return (below / (below + above)) * 100;
}

function parseQty(raw: string | undefined): number | null {
  if (!raw) return null;
  const n = Number(String(raw).replace(/,/g, '').replace(/^\+/, '').trim());
  return Number.isFinite(n) ? n : null;
}

function isFiniteNumber(value: number | undefined): value is number {
  return value != null && Number.isFinite(value);
}
