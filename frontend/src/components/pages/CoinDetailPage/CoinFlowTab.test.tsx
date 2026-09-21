import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CoinFlowTab } from './CoinFlowTab';

const empty = { data: undefined, currentData: undefined, isError: false, isFetching: false, isLoading: false };

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
    useGetMarketFlowAtPricesQuery: hook,
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
});
