import type { SpotMarket, SpotSortField } from '@/libs/api';

/** Table surfaces that share the spot metric catalog. */
export type SpotMetricSurface = 'markets' | 'watchlist';

/**
 * Metric column ids. Add a new entry here (and in SPOT_METRICS) to expose it
 * in Markets and/or Watchlist via the column picker.
 */
export type SpotMetricId =
  | 'lastPrice'
  | 'priceChangePercent'
  | 'priceChange'
  | 'highPrice'
  | 'lowPrice'
  | 'volume'
  | 'quoteVolume'
  | 'marketCapCirculating'
  | 'marketCapTotal'
  | 'marketCapMax'
  | 'tradeCount'
  | 'tags';

export type SpotMetricFormat =
  | 'price'
  | 'changePercent'
  | 'compactUsd'
  | 'tradeCount'
  | 'tags'
  | 'number';

export type SpotMetricDef = {
  id: SpotMetricId;
  /** Field on SpotMarket used for display (and dataIndex on Markets). */
  field: keyof SpotMarket;
  format: SpotMetricFormat;
  /** When set, Markets table can sort by this API field. */
  sortField?: SpotSortField;
  /** i18n key under markets:table.* */
  labelKey: string;
  surfaces: readonly SpotMetricSurface[];
  defaultVisible: Partial<Record<SpotMetricSurface, boolean>>;
  align?: 'left' | 'right';
  /** Color the value with changeTone (24h %). */
  toneFromChange?: boolean;
};

/**
 * Single catalog of spot metrics for Markets dashboard + Watchlist.
 * To add a column: append a def, add i18n labels, ship — no table rewrites.
 */
export const SPOT_METRICS: readonly SpotMetricDef[] = [
  {
    id: 'lastPrice',
    field: 'lastPrice',
    format: 'price',
    sortField: 'lastPrice',
    labelKey: 'last',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: true, watchlist: true },
    align: 'right',
  },
  {
    id: 'priceChangePercent',
    field: 'priceChangePercent',
    format: 'changePercent',
    sortField: 'priceChangePercent',
    labelKey: 'change24h',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: true, watchlist: true },
    align: 'right',
    toneFromChange: true,
  },
  {
    id: 'priceChange',
    field: 'priceChange',
    format: 'price',
    labelKey: 'changeAbs',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
    toneFromChange: true,
  },
  {
    id: 'highPrice',
    field: 'highPrice',
    format: 'price',
    labelKey: 'high',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
  },
  {
    id: 'lowPrice',
    field: 'lowPrice',
    format: 'price',
    labelKey: 'low',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
  },
  {
    id: 'volume',
    field: 'volume',
    format: 'compactUsd',
    sortField: 'volume',
    labelKey: 'baseVol',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
  },
  {
    id: 'quoteVolume',
    field: 'quoteVolume',
    format: 'compactUsd',
    sortField: 'quoteVolume',
    labelKey: 'quoteVol',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: true, watchlist: true },
    align: 'right',
  },
  {
    id: 'marketCapCirculating',
    field: 'marketCapCirculating',
    format: 'compactUsd',
    sortField: 'marketCapCirculating',
    labelKey: 'circMcap',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: true, watchlist: true },
    align: 'right',
  },
  {
    id: 'marketCapTotal',
    field: 'marketCapTotal',
    format: 'compactUsd',
    sortField: 'marketCapTotal',
    labelKey: 'totalMcap',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
  },
  {
    id: 'marketCapMax',
    field: 'marketCapMax',
    format: 'compactUsd',
    sortField: 'marketCapMax',
    labelKey: 'maxMcap',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: false, watchlist: false },
    align: 'right',
  },
  {
    id: 'tradeCount',
    field: 'tradeCount',
    format: 'tradeCount',
    sortField: 'tradeCount',
    labelKey: 'trades',
    surfaces: ['markets', 'watchlist'],
    defaultVisible: { markets: true, watchlist: false },
    align: 'right',
  },
  {
    id: 'tags',
    field: 'tags',
    format: 'tags',
    sortField: 'tags',
    labelKey: 'tags',
    surfaces: ['markets'],
    defaultVisible: { markets: true },
    align: 'left',
  },
] as const;

const BY_ID: Record<SpotMetricId, SpotMetricDef> = SPOT_METRICS.reduce(
  (acc, m) => {
    acc[m.id] = m;
    return acc;
  },
  {} as Record<SpotMetricId, SpotMetricDef>,
);

export function getSpotMetric(id: SpotMetricId): SpotMetricDef {
  return BY_ID[id];
}

export function metricsForSurface(surface: SpotMetricSurface): SpotMetricDef[] {
  return SPOT_METRICS.filter((m) => m.surfaces.includes(surface));
}

export function defaultMetricIds(surface: SpotMetricSurface): SpotMetricId[] {
  return metricsForSurface(surface)
    .filter((m) => m.defaultVisible[surface])
    .map((m) => m.id);
}

/** Ant column key → API sort field (only metrics that support sort). */
export function columnSortMap(): Record<string, SpotSortField> {
  const map: Record<string, SpotSortField> = { symbol: 'symbol' };
  for (const m of SPOT_METRICS) {
    if (m.sortField) map[m.id] = m.sortField;
  }
  return map;
}

const STORAGE_PREFIX = 'swyngora.metricColumns.';

function storageKey(surface: SpotMetricSurface): string {
  return `${STORAGE_PREFIX}${surface}`;
}

/** Validate and order ids against the catalog for a surface. */
export function normalizeMetricIds(
  surface: SpotMetricSurface,
  ids: readonly string[] | null | undefined,
): SpotMetricId[] {
  const allowed = new Set(metricsForSurface(surface).map((m) => m.id));
  const out: SpotMetricId[] = [];
  const seen = new Set<SpotMetricId>();
  for (const raw of ids ?? []) {
    const id = raw as SpotMetricId;
    if (!allowed.has(id) || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out.length > 0 ? out : defaultMetricIds(surface);
}

export function loadMetricIds(surface: SpotMetricSurface): SpotMetricId[] {
  if (typeof localStorage === 'undefined') return defaultMetricIds(surface);
  try {
    const raw = localStorage.getItem(storageKey(surface));
    if (!raw) return defaultMetricIds(surface);
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return defaultMetricIds(surface);
    return normalizeMetricIds(
      surface,
      parsed.filter((x): x is string => typeof x === 'string'),
    );
  } catch {
    return defaultMetricIds(surface);
  }
}

export function saveMetricIds(surface: SpotMetricSurface, ids: readonly SpotMetricId[]): void {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(storageKey(surface), JSON.stringify(normalizeMetricIds(surface, ids)));
  } catch {
    // ignore quota / private mode
  }
}

/** Ordered metric defs for the current selection. */
export function resolveMetricDefs(
  surface: SpotMetricSurface,
  ids: readonly SpotMetricId[],
): SpotMetricDef[] {
  return normalizeMetricIds(surface, ids).map((id) => BY_ID[id]);
}
