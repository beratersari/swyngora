import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { BreadthOverview } from './BreadthOverview';

describe('BreadthOverview', () => {
  it('shows 1h up vs down share', () => {
    renderWithTheme(
      <BreadthOverview
        data={{
          universe: 80,
          summary: 'Most coins are up with BTC.',
          windows: [
            {
              window: '1h',
              alignment: 'with_market',
              counts: { upPct: '62', downPct: '28' },
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('62% up · 28% down')).toBeInTheDocument();
    expect(screen.getByText('Most coins are up with BTC.')).toBeInTheDocument();
  });

  it('lists extra windows besides the 1h headline', () => {
    renderWithTheme(
      <BreadthOverview
        data={{
          windows: [
            { window: '1h', counts: { upPct: '62', downPct: '28' } },
            { window: '4h', counts: { upPct: '55', downPct: '40' } },
          ],
        }}
      />,
    );
    expect(screen.getByText('4h')).toBeInTheDocument();
    expect(screen.getByText('55% up · 40% down')).toBeInTheDocument();
  });
});
