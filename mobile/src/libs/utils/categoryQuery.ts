import type { MarketExchange, SpotListQuery, SpotSortField, SpotSortOrder } from '@/libs/api';
import { FEATURED_CATEGORY_TAGS } from '@/config/categoryConstants';

export type CategorySpotParamsInput = {
  tag: string;
  exchange?: MarketExchange | string;
  quote?: string;
  sort?: SpotSortField;
  order?: SpotSortOrder;
  limit?: number;
  offset?: number;
};

/**
 * Keep featured order; only include tags present in the live catalog (case-insensitive match, preserve live casing).
 */
export function intersectFeaturedTags(
  liveTags: string[] | undefined | null,
  featured: readonly string[] = FEATURED_CATEGORY_TAGS,
): string[] {
  if (!liveTags?.length) return [];
  const byLower = new Map<string, string>();
  for (const t of liveTags) {
    const key = t.trim().toLowerCase();
    if (key && !byLower.has(key)) byLower.set(key, t.trim());
  }
  const out: string[] = [];
  for (const f of featured) {
    const live = byLower.get(f.trim().toLowerCase());
    if (live) out.push(live);
  }
  return out;
}

/** Case-insensitive substring filter; preserves input order. */
export function filterTagsBySearch(
  tags: string[] | undefined | null,
  query: string,
): string[] {
  if (!tags?.length) return [];
  const q = query.trim().toLowerCase();
  if (!q) return [...tags];
  return tags.filter((t) => t.toLowerCase().includes(q));
}

/** Display label for API tag ids (underscores → spaces). Prefer i18n when available. */
export function formatCategoryLabel(tag: string): string {
  return tag.replace(/_/g, ' ').trim();
}

/**
 * Spot list query for a single discovery tag.
 * Omits empty tag (caller should not request without a tag for discovery).
 */
export function buildCategorySpotParams(input: CategorySpotParamsInput): SpotListQuery {
  const tag = input.tag.trim();
  return {
    exchange: (input.exchange as MarketExchange | undefined) ?? 'binance',
    quote: input.quote || 'USDT',
    tags: tag || undefined,
    sort: input.sort ?? 'quoteVolume',
    order: input.order ?? 'desc',
    limit: input.limit ?? 30,
    offset: input.offset ?? 0,
    status: 'TRADING',
  };
}
