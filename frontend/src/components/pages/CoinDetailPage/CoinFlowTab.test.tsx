import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { CoinFlowTab } from './CoinFlowTab';

const empty = { data: undefined, currentData: undefined, isError: false, isFetching: false, isLoading: false };
const flowWindows: string[] = [];

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  const hook = () => empty;
  const perp = {
    summary: '40M bid vs 15M ask',
    combined: {
      headline: { below: { notional: '40000000' }, above: { notional: '15000000' }, lean: 'bid' },
    },
    venues: [
      { exchange: 'binance', headline: { below: { notional: '20000000' }, above: { notional: '8000000' } } },
      { exchange: 'okx', headline: { below: { notional: '12000000' }, above: { notional: '4000000' } } },
    ],
  };
  return {
    ...actual,
    useGetMarketPerpDepthQuery: () => ({
      ...empty,
      data: perp,
      currentData: perp,
    }),
    useGetMarketVolumeSurgeQuery: hook,
    useGetMarketTakerFlowQuery: () => ({
      ...empty,
      data: {
        summary: 'Buyers in control',
        venues: [{ exchange: 'okx', summary: 'OKX buyers led the newest prints' }],
      },
      currentData: {
        summary: 'Buyers in control',
        venues: [{ exchange: 'okx', summary: 'OKX buyers led the newest prints' }],
      },
    }),
    useGetMarketFlowAtPricesQuery: (arg: { window?: string }) => {
      flowWindows.push(arg.window ?? '');
      const report = {
        summary: 'Most prints at 65520.',
        combined: {
          levels: [
            {
              price: '65520',
              buyVolume: '90000',
              sellVolume: '10000',
              volume: '100000',
              delta: '80000',
              sharePct: '50',
              trades: 4,
              isPoc: true,
            },
          ],
        },
      };
      return { ...empty, data: report, currentData: report };
    },
    useGetMarketVWAPQuery: hook,
    useGetMarketVolumeProfileQuery: hook,
    useGetMarketAbsorptionQuery: hook,
    useGetMarketLiquiditySweepsQuery: hook,
    useGetSweepRecoveryQuery: () => ({
      ...empty,
      data: {
        summary: 'Typical refill was 5m.',
        rows: [{ exchange: 'binance', side: 'high', level: '65000', recovered: true, recoverySeconds: 300 }],
      },
      currentData: {
        summary: 'Typical refill was 5m.',
        rows: [{ exchange: 'binance', side: 'high', level: '65000', recovered: true, recoverySeconds: 300 }],
      },
    }),
    useGetMarketWhalesQuery: hook,
    useGetMarketIcebergsQuery: hook,
    useGetMarketBookLiquidityFlowQuery: hook,
  };
});

describe('CoinFlowTab', () => {
  it('shows perp depth and taker summaries', () => {
    renderWithProviders(<CoinFlowTab symbol="BTCUSDT" exchange="binance" />);
    expect(screen.getByText('40M bid vs 15M ask')).toBeInTheDocument();
    expect(screen.getByText('okx')).toBeInTheDocument();
    expect(screen.getByText('Buyers in control')).toBeInTheDocument();
    expect(screen.getByText('OKX buyers led the newest prints')).toBeInTheDocument();
    expect(screen.getByText('Most prints at 65520.')).toBeInTheDocument();
    expect(screen.getByText('Typical refill was 5m.')).toBeInTheDocument();
    expect(screen.getByText(/65000/)).toHaveTextContent('5m');
    expect(screen.getAllByText('POC').length).toBeGreaterThan(0);
    expect(screen.getByText(/9\.0× buy/i)).toBeInTheDocument();
  });

  it('asks for a longer flow-at-prices window', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CoinFlowTab symbol="BTCUSDT" exchange="binance" />);
    await user.click(screen.getByRole('radio', { name: '1d' }));
    expect(flowWindows.at(-1)).toBe('1d');
  });
});
