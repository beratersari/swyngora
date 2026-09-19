import { describe, it, expect } from 'vitest';
import {
  buildCategorySpotParams,
  filterTagsBySearch,
  formatCategoryLabel,
  intersectFeaturedTags,
} from './categoryQuery';

describe('intersectFeaturedTags', () => {
  it('preserves featured order and live casing', () => {
    const live = ['defi', 'Meme', 'Other', 'AI'];
    expect(intersectFeaturedTags(live, ['Meme', 'AI', 'defi', 'Missing'])).toEqual([
      'Meme',
      'AI',
      'defi',
    ]);
  });

  it('returns empty when live is empty', () => {
    expect(intersectFeaturedTags([])).toEqual([]);
    expect(intersectFeaturedTags(null)).toEqual([]);
  });

  it('matches case-insensitively', () => {
    expect(intersectFeaturedTags(['meme', 'ai'], ['Meme', 'AI'])).toEqual(['meme', 'ai']);
  });
});

describe('filterTagsBySearch', () => {
  it('filters case-insensitively', () => {
    expect(filterTagsBySearch(['Meme', 'defi', 'AI'], 'me')).toEqual(['Meme']);
  });

  it('returns all when query empty', () => {
    expect(filterTagsBySearch(['a', 'b'], '  ')).toEqual(['a', 'b']);
  });
});

describe('formatCategoryLabel', () => {
  it('replaces underscores', () => {
    expect(formatCategoryLabel('Layer1_Layer2')).toBe('Layer1 Layer2');
  });
});

describe('buildCategorySpotParams', () => {
  it('builds single-tag spot query', () => {
    expect(buildCategorySpotParams({ tag: 'Meme' })).toEqual({
      exchange: 'binance',
      quote: 'USDT',
      tags: 'Meme',
      sort: 'quoteVolume',
      order: 'desc',
      limit: 30,
      offset: 0,
      status: 'TRADING',
    });
  });

  it('omits empty tag', () => {
    expect(buildCategorySpotParams({ tag: '  ' }).tags).toBeUndefined();
  });
});
