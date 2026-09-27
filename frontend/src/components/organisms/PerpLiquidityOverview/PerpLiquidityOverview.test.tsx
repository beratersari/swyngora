import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PerpLiquidityOverview } from './PerpLiquidityOverview';

describe('PerpLiquidityOverview', () => {
  it('shows a bid versus ask bar and keeps bands and windows in the dialog', async () => {
    const user = userEvent.setup();
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
          bands: [
            { rangePct: '0.1', total: '80000000', ofHeadlinePct: '8', leaders: ['ETH'] },
            { rangePct: '1', total: '4200000000', ofHeadlinePct: '100', leaders: ['BTC'] },
          ],
          windows: [
            {
              window: '1h',
              complete: true,
              pastBidPct: '55',
              bidPct: '65',
              changePct: '+10',
              pastTotal: '400000000',
              total: '500000000',
              totalChangePct: '+25',
              depthKind: 'thicker',
              drivers: [{ base: 'ETH', changePct: '+10' }],
            },
          ],
        }}
      />,
    );
    expect(screen.getByRole('button', { name: /Perp bid \/ ask/ })).toBeInTheDocument();
    expect(screen.getByText('Bid-heavy')).toBeInTheDocument();
    expect(screen.getByText('2.40B / 1.80B')).toBeInTheDocument();
    expect(screen.queryByText('55% → 65% (+10)')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Perp bid \/ ask/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('48%')).toBeInTheDocument();
    expect(screen.getByText('55% → 65% (+10)')).toBeInTheDocument();
    expect(screen.getByText('400.00M → 500.00M (+25%) · Thicker')).toBeInTheDocument();
    expect(screen.getByText('80.00M · 8%')).toBeInTheDocument();
  });
});
