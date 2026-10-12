import { describe, expect, it } from 'vitest';
import { formatRecordPnl } from './helpers';

describe('formatRecordPnl', () => {
  it('keeps a plus on a gain and a minus on a loss', () => {
    expect(formatRecordPnl(12.5).startsWith('+')).toBe(true);
    expect(formatRecordPnl(-4).startsWith('-')).toBe(true);
    expect(formatRecordPnl(0)).toBe('0');
    expect(formatRecordPnl(Number.NaN)).toBe('—');
  });
});
