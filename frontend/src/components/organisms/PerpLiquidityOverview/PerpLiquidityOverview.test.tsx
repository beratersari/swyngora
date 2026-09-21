import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PerpLiquidityOverview } from './PerpLiquidityOverview';

describe('PerpLiquidityOverview', () => {
  it('shows total bid/ask and leading coin shares', () => {
    renderWithTheme(
      <PerpLiquidityOverview
        data={{
          exchange: 'all',
          rangePct: '1',
          known: 2,
          bid: '2400000000',
          ask: '1800000000',
          lean: 'bid',
          summary: '2 liquid perps: 2.4B bid vs 1.8B ask within 1% of mid (bid-heavy). BTC 48% and ETH 17% of that book.',
          coins: [
            { symbol: 'BTCUSDT', base: 'BTC', bid: '1800000000', ask: '900000000', total: '2700000000', sharePct: '48' },
            { symbol: 'ETHUSDT', base: 'ETH', bid: '600000000', ask: '900000000', total: '1500000000', sharePct: '17' },
          ],
          leaders: ['BTC', 'ETH'],
          windows: [
            {
              window: '1h',
              complete: true,
              pastBidPct: '55',
              bidPct: '65',
              changePct: '+10',
              drivers: [{ base: 'ETH', changePct: '+10' }],
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('2.40B')).toBeInTheDocument();
    expect(screen.getByText('1.80B')).toBeInTheDocument();
    expect(screen.getByText('48%')).toBeInTheDocument();
    expect(screen.getByText('BTC')).toBeInTheDocument();
    expect(screen.getByText('55% → 65% (+10)')).toBeInTheDocument();
  });
});
