import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CoinStructureTab } from './CoinStructureTab';

const empty = { data: undefined, currentData: undefined, isError: false, isFetching: false, isLoading: false };

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  const hook = () => empty;
  return {
    ...actual,
    useGetMarketCorrelationQuery: () => ({
      ...empty,
      data: { summary: 'SOL follows BTC (0.82) over 4h' },
      currentData: { summary: 'SOL follows BTC (0.82) over 4h' },
    }),
    useGetMarketVolatilityQuery: hook,
    useGetMarketLevelsQuery: () => ({
      ...empty,
      data: { summary: 'POC 94.2k' },
      currentData: { summary: 'POC 94.2k' },
    }),
    useGetMarketSqueezeRiskQuery: hook,
    useGetMarketPositioningQuery: hook,
    useGetMarketVenueDivergenceQuery: hook,
  };
});

describe('CoinStructureTab', () => {
  it('shows correlation and levels summaries', () => {
    renderWithProviders(<CoinStructureTab symbol="SOLUSDT" exchange="binance" />);
    expect(screen.getByText('SOL follows BTC (0.82) over 4h')).toBeInTheDocument();
    expect(screen.getByText('POC 94.2k')).toBeInTheDocument();
  });
});
