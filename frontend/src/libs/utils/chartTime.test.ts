import { describe, expect, it } from 'vitest';
import { formatChartDateTime } from './chartTime';

describe('formatChartDateTime', () => {
  it('formats unix seconds with hour, minute, and second', () => {
    const label = formatChartDateTime(1_704_067_261);
    expect(label).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  });

  it('formats a business-day object as midnight local', () => {
    expect(formatChartDateTime({ year: 2026, month: 8, day: 22 })).toMatch(
      /^2026-08-22 \d{2}:\d{2}:\d{2}$/,
    );
  });
});
