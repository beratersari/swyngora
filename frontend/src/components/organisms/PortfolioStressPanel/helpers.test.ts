import { describe, expect, it } from 'vitest';
import { equityChangePct, portfolioChange } from './helpers';

describe('portfolio change', () => {
  it('puts the percent of current equity beside the money change', () => {
    expect(equityChangePct(10000, -105)).toBeCloseTo(-1.05);
    expect(equityChangePct(10000, 250)).toBeCloseTo(2.5);
    const line = portfolioChange(10000, -200);
    expect(line).toMatch(/200/);
    expect(line).toMatch(/2\s*%/);
  });

  it('leaves the percent off when equity is not a positive base', () => {
    expect(equityChangePct(0, -10)).toBeUndefined();
    expect(equityChangePct(-100, 10)).toBeUndefined();
    expect(portfolioChange(0, -10)).not.toMatch(/%/);
  });
});
