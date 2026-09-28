import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PerpLiquidityOverview } from './PerpLiquidityOverview';

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries: () => ({ setData: vi.fn() }),
    timeScale: () => ({ fitContent: vi.fn() }),
    applyOptions: vi.fn(),
    remove: vi.fn(),
  })),
  LineSeries: {},
}));

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', ResizeObserverStub);

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
              sizeDrivers: [
                { base: 'ETH', change: '+100000000' },
                { base: 'BTC', change: '+20000000' },
              ],
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
    expect(screen.getByText('ETH +100.00M, BTC +20.00M')).toBeInTheDocument();
    expect(screen.getByText('80.00M · 8%')).toBeInTheDocument();
    expect(screen.getByText('The path shows up after this desk has been sampling.')).toBeInTheDocument();
  });

  it('draws the six-hour total once two minutes exist', async () => {
    const user = userEvent.setup();
    renderWithTheme(
      <PerpLiquidityOverview
        data={{
          bid: '100',
          ask: '100',
          history: [
            { time: '2026-09-21T14:00:00Z', total: '400', price: '100' },
            { time: '2026-09-21T16:00:00Z', total: '900', price: '90' },
          ],
        }}
      />,
    );
    await user.click(screen.getByRole('button', { name: /Perp bid \/ ask/ }));
    expect(await screen.findByTestId('perp-depth-history')).toBeInTheDocument();
    expect(screen.getByText('BTC')).toBeInTheDocument();
    expect(screen.getByText('Book')).toBeInTheDocument();
    expect(screen.getByText('No jump or drop of 5% inside 15 minutes yet.')).toBeInTheDocument();
  });

  it('lists the biggest book jump with BTC during and after', async () => {
    const user = userEvent.setup();
    renderWithTheme(
      <PerpLiquidityOverview
        data={{
          bid: '100',
          ask: '100',
          history: [
            { time: '2026-09-21T14:00:00Z', total: '400000000', price: '100000' },
            { time: '2026-09-21T14:06:00Z', total: '900000000', price: '99000' },
          ],
          moves: [
            {
              kind: 'jump',
              from: '2026-09-21T14:00:00Z',
              to: '2026-09-21T14:06:00Z',
              pastTotal: '400000000',
              total: '900000000',
              changePct: '+125',
              priceFrom: '100000',
              priceTo: '99000',
              priceChangePct: '-1',
              priceAfter: '99500',
              priceAfterPct: '+0.51',
              span: 'one',
              drivers: [{ base: 'ETH', change: '+500000000', sharePct: '+83' }],
            },
          ],
        }}
      />,
    );
    await user.click(screen.getByRole('button', { name: /Perp bid \/ ask/ }));
    expect(await screen.findByText('Biggest book moves')).toBeInTheDocument();
    expect(screen.getByText('Jump')).toBeInTheDocument();
    expect(screen.getByText('400.00M → 900.00M (+125%)')).toBeInTheDocument();
    expect(screen.getByText('BTC 100.00K → 99.00K (-1%)')).toBeInTheDocument();
    expect(screen.getByText('15m later 99.50K (+0.51%)')).toBeInTheDocument();
    expect(screen.getByText('ETH +500.00M (+83%) · One coin')).toBeInTheDocument();
  });
});
