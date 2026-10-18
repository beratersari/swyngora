import { describe, expect, it } from 'vitest';
import { lossLine } from './helpers';

describe('lossLine', () => {
  it('shows a loss with a minus and a gain with a plus', () => {
    expect(lossLine(-8.2)).toBe('-8.20%');
    expect(lossLine(1.5)).toBe('+1.50%');
    expect(lossLine(0)).toBe('0.00%');
    expect(lossLine(undefined)).toBe('—');
  });
});
