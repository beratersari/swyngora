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
        termStructure: [
          { label: '27 Mar 2026', atmIv: '51', callIv: '50', putIv: '52', skew: '2' },
        ],
        skew: { atmSkew: '2', lean: 'puts', termSlope: '-1' },
        gamma: {
          netUsd: '-840000000',
          callUsd: '-1200000000',
          putUsd: '360000000',
          regime: 'short',
          flip: '97200',
          flipSide: 'above',
        },
        expiries: [{
          expiry: '2026-03-27T08:00:00Z',
          label: '27 Mar 2026',
          daysToExpiry: 26,
          sharePct: '80',
          openInterestUsd: '8000000',
          callOi: '8',
          putOi: '2',
          expectedMovePct: '13.6',
          expectedMoveAbs: '13600',
          expectedLow: '86400',
          expectedHigh: '113600',
          maxPain: '100000',
        }],
        moves: {
          findings: [{
            kind: 'oi_jump',
            topic: 'oi',
            window: '1h',
            title: 'Open interest jumped 14% in 1h — well above typical',
          }],
        },
        vsMarket: {
          compared: true,
          findings: [{
            kind: 'puts_vs_futures_long',
            topic: 'lean',
            title: 'Options lean put while futures accounts are long',
          }],
        },
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
        termStructure: [
          { label: '27 Mar 2026', atmIv: '51', callIv: '50', putIv: '52', skew: '2' },
        ],
        skew: { atmSkew: '2', lean: 'puts', termSlope: '-1' },
        gamma: {
          netUsd: '-840000000',
          callUsd: '-1200000000',
          putUsd: '360000000',
          regime: 'short',
          flip: '97200',
          flipSide: 'above',
        },
        expiries: [{
          expiry: '2026-03-27T08:00:00Z',
          label: '27 Mar 2026',
          daysToExpiry: 26,
          sharePct: '80',
          openInterestUsd: '8000000',
          callOi: '8',
          putOi: '2',
          expectedMovePct: '13.6',
          expectedMoveAbs: '13600',
          expectedLow: '86400',
          expectedHigh: '113600',
          maxPain: '100000',
        }],
        moves: {
          findings: [{
            kind: 'oi_jump',
            topic: 'oi',
            window: '1h',
            title: 'Open interest jumped 14% in 1h — well above typical',
          }],
        },
        vsMarket: {
          compared: true,
          findings: [{
            kind: 'puts_vs_futures_long',
            topic: 'lean',
            title: 'Options lean put while futures accounts are long',
          }],
        },
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
    expect(screen.getAllByText('27 Mar 2026').length).toBeGreaterThan(0);
    expect(screen.getByText(/volatility structure/i)).toBeInTheDocument();
    expect(screen.getAllByText(/dealer gamma/i).length).toBeGreaterThan(0);
    expect(screen.getByText('−840.00M')).toBeInTheDocument();
    expect(screen.getByText('±13.6%')).toBeInTheDocument();
    expect(screen.getByText(/important options moves/i)).toBeInTheDocument();
    expect(screen.getByText(/open interest jumped 14%/i)).toBeInTheDocument();
    expect(screen.getByText(/over time/i)).toBeInTheDocument();
    expect(screen.getByText(/options vs the rest of the market/i)).toBeInTheDocument();
    expect(screen.getByText(/lean put while futures accounts are long/i)).toBeInTheDocument();
  });
});
