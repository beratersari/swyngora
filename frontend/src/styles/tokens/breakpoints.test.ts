import { describe, expect, it } from 'vitest';
import { breakpoints, media, mediaQueries } from './breakpoints';

describe('breakpoints', () => {
  it('exposes phone and tablet cutoffs used by the desk', () => {
    expect(breakpoints.phone).toBe(720);
    expect(breakpoints.tablet).toBe(960);
    expect(media.phone).toContain('719.98px');
    expect(mediaQueries.phone).toContain('719.98px');
  });
});
