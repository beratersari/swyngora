import { describe, expect, it } from 'vitest';
import { splitWidths } from './helpers';

describe('splitWidths', () => {
  it('scales the parts so they fill the bar', () => {
    expect(
      splitWidths([
        { key: 'bid', pct: 240, color: '#16C784' },
        { key: 'ask', pct: 160, color: '#EA3943' },
      ]),
    ).toEqual(['60%', '40%']);
  });

  it('drops a missing side instead of painting a full bar', () => {
    expect(splitWidths([{ key: 'up', pct: 0, color: '#16C784' }])).toEqual(['0%']);
  });
});
