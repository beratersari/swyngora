import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ArbPage } from './ArbPage';

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useLazyScanPriceDiffQuotesQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
    useListPriceDiffWatchesQuery: () => ({ data: { watches: [] } }),
    useListPriceDiffOpportunitiesQuery: () => ({ data: { opportunities: [] }, isFetching: false }),
    useGetPriceDiffOpportunityQuery: () => ({ data: undefined, isFetching: false }),
    useCreatePriceDiffWatchMutation: () => [vi.fn(), { isLoading: false }],
    useDeletePriceDiffWatchMutation: () => [vi.fn()],
    useLazyScanMarketFundingArbQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
    useListFundingArbWatchesQuery: () => ({ data: { watches: [] } }),
    useListFundingArbSignalsQuery: () => ({ data: { signals: [] } }),
    useCreateFundingArbWatchMutation: () => [vi.fn(), { isLoading: false }],
    useDeleteFundingArbWatchMutation: () => [vi.fn()],
  };
});

describe('ArbPage', () => {
  it('renders the price-gap desk by default', () => {
    renderWithProviders(<ArbPage />);
    expect(screen.getByText(/cross-exchange|çapraz borsa/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /scan books|kitapları tara/i })).toBeInTheDocument();
  });
});
