/** Featured product-catalog tags for Home / Categories (intersect with live /tags). */

export const FEATURED_CATEGORY_TAGS = [
  'Meme',
  'AI',
  'defi',
  'Layer1_Layer2',
  'Payments',
] as const;

/** Catalog source for tag list — Binance marketing tags only. */
export const CATEGORY_TAGS_EXCHANGE = 'binance' as const;

export const CATEGORY_TAG_SEARCH_DEBOUNCE_MS = 200;
