import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsPage } from './OptionsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketOptionsQuery: () => ({
      data: {
        asset: 'BTC',
        exchange: 'deribit',
        openInterest: '350',
        openInterestUsd: '35000000',
        volume: '12',
        volumeUsd: '1200000',
        putCallOi: '0.4',
        lean: 'calls',
        dvol: '49.5',
        expiries: [{ label: '27 Mar 2026', sharePct: '80', openInterestUsd: '8000000', callOi: '8', putOi: '2' }],
        note: 'Informational only.',
      },
      currentData: {
        asset: 'BTC',
        exchange: 'deribit',
        openInterest: '350',
        openInterestUsd: '35000000',
        volume: '12',
        volumeUsd: '1200000',
        putCallOi: '0.4',
        lean: 'calls',
        dvol: '49.5',
        expiries: [{ label: '27 Mar 2026', sharePct: '80', openInterestUsd: '8000000', callOi: '8', putOi: '2' }],
        note: 'Informational only.',
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

describe('OptionsPage', () => {
  it('renders the options desk', () => {
    renderWithProviders(<OptionsPage />, { routerEntries: ['/options'] });
    expect(screen.getByRole('heading', { name: /options/i })).toBeInTheDocument();
    expect(screen.getByText('27 Mar 2026')).toBeInTheDocument();
  });
});
