import { describe, expect, it } from 'vitest';
import { resolveSkeletonStyle } from './Skeleton.helpers';

describe('resolveSkeletonStyle', () => {
  it('uses defaults per variant', () => {
    const chart = resolveSkeletonStyle('chart');
    expect(chart.width).toBeDefined();
    expect(chart.height).toBeDefined();
  });

  it('overrides width and height', () => {
    const s = resolveSkeletonStyle('button', 120, 40);
    expect(s.width).toBe(120);
    expect(s.height).toBe(40);
  });
});
