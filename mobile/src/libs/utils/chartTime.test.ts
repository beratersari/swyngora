import { describe, expect, it } from 'vitest';
import { formatChartDateTime } from './chartTime';

describe('formatChartDateTime', () => {
  it('includes hour, minute, and second', () => {
    expect(formatChartDateTime(1_704_067_261)).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  });
});
