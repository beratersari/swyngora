import { describe, expect, it } from 'vitest';
import {
  formatHolderAddress,
  formatHolderBalance,
  formatHolderCount,
  formatSharePct,
} from './helpers';

describe('formatHolderAddress', () => {
  it('keeps short addresses', () => {
    expect(formatHolderAddress('34xp4v')).toBe('34xp4v');
  });

  it('truncates long addresses', () => {
    expect(formatHolderAddress('34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo')).toBe('34xp4v…wseo');
  });
});

describe('formatSharePct', () => {
  it('formats and falls back', () => {
    expect(formatSharePct(5.4)).toBe('5.4%');
    expect(formatSharePct(null)).toBe('—');
  });
});

describe('formatHolderCount', () => {
  it('compacts large counts', () => {
    expect(formatHolderCount(50_708_169)).toBe('50.71M');
    expect(formatHolderCount(undefined)).toBe('—');
  });
});

describe('formatHolderBalance', () => {
  it('does not round dust to zero', () => {
    expect(formatHolderBalance(0.0004123)).not.toBe('0');
    expect(formatHolderBalance(0.0004123)).toMatch(/0\.00041/);
  });

  it('compacts large stacks', () => {
    expect(formatHolderBalance(1_250_000_000)).toBe('1.25B');
  });
});

