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
      data: { summary: 'Buyers in control' },
      currentData: { summary: 'Buyers in control' },
    }),
    useGetMarketFlowAtPricesQuery: (arg: { window?: string }) => {
      flowWindows.push(arg.window ?? '');
      return empty;
    },
    useGetMarketVWAPQuery: hook,
    useGetMarketVolumeProfileQuery: hook,
    useGetMarketAbsorptionQuery: hook,
    useGetMarketLiquiditySweepsQuery: hook,
    useGetMarketWhalesQuery: hook,
    useGetMarketIcebergsQuery: hook,
    useGetMarketBookLiquidityFlowQuery: hook,
  };
});

describe('CoinFlowTab', () => {
  it('shows perp depth and taker summaries', () => {
    renderWithProviders(<CoinFlowTab symbol="BTCUSDT" exchange="binance" />);
    expect(screen.getByText('40M bid vs 15M ask')).toBeInTheDocument();
    expect(screen.getByText('Buyers in control')).toBeInTheDocument();
  });

  it('asks for a longer flow-at-prices window', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CoinFlowTab symbol="BTCUSDT" exchange="binance" />);
    await user.click(screen.getByRole('radio', { name: '1d' }));
    expect(flowWindows.at(-1)).toBe('1d');
  });
});
