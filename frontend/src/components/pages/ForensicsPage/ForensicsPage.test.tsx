import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsPage } from './ForensicsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanMarketForensicsQuery: () => ({
      data: {
        summary: '1 of 4 scanned coin(s) currently look like a past pre-move setup.',
        hits: [
          {
            symbol: 'ETHUSDT',
            similarity: '77',
            lean: 'up',
            afterStory: 'When ETH looked like this before, price more often kept rising over the next 1h.',
          },
        ],
      },
      currentData: {
        summary: '1 of 4 scanned coin(s) currently look like a past pre-move setup.',
        hits: [
          {
            symbol: 'ETHUSDT',
            similarity: '77',
            lean: 'up',
            afterStory: 'When ETH looked like this before, price more often kept rising over the next 1h.',
          },
        ],
      },
      isLoading: false,
      isError: false,
    }),
    useGetMarketForensicsQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        summary: 'BTC rose on unusually heavy trading.',
        note: 'Informational only',
        why: {
          headline: 'BTC rose on unusually heavy trading.',
          before: 'Bids had already thinned, so there was less support under the market.',
          during: 'Open interest fell as price rose — a short-cover shape.',
          after: 'After the move, BTC went quiet.',
          similar: 'When the tape looked like this before, BTC more often kept rising over the next 1h.',
        },
        tape: {
          combined: {
            exchange: 'combined',
            summary: 'Combined tape',
            phases: [
              { phase: 'before', complete: true, price: { close: '100' } },
              { phase: 'during', complete: true, price: { close: '107', changePct: '+7%' } },
              { phase: 'after', complete: true, price: { close: '105' } },
            ],
            changes: [{ metric: 'price', before: '100', during: '107', after: '105', path: 'faded' }],
          },
        },
        similar: {
          summary: '2 similar past setups',
          matches: [
            {
              at: '2026-08-18T12:00:00Z',
              until: '2026-08-18T12:15:00Z',
              direction: 'up',
              similarity: '82',
              coverage: '90',
              afterReturnPct: '+3.5',
            },
          ],
          afterHorizons: [{ horizon: '1h', up: 2, down: 0, medianPct: '+2.1', sample: 2 }],
        },
      },
      currentData: {
        symbol: 'BTCUSDT',
        summary: 'BTC rose on unusually heavy trading.',
        note: 'Informational only',
        why: {
          headline: 'BTC rose on unusually heavy trading.',
          before: 'Bids had already thinned, so there was less support under the market.',
          during: 'Open interest fell as price rose — a short-cover shape.',
          after: 'After the move, BTC went quiet.',
          similar: 'When the tape looked like this before, BTC more often kept rising over the next 1h.',
        },
        tape: {
          combined: {
            exchange: 'combined',
            summary: 'Combined tape',
            phases: [
              { phase: 'before', complete: true, price: { close: '100' } },
              { phase: 'during', complete: true, price: { close: '107', changePct: '+7%' } },
              { phase: 'after', complete: true, price: { close: '105' } },
            ],
            changes: [{ metric: 'price', before: '100', during: '107', after: '105', path: 'faded' }],
          },
        },
        similar: {
          summary: '2 similar past setups',
          matches: [
            {
              at: '2026-08-18T12:00:00Z',
              until: '2026-08-18T12:15:00Z',
              direction: 'up',
              similarity: '82',
              coverage: '90',
              afterReturnPct: '+3.5',
            },
          ],
          afterHorizons: [{ horizon: '1h', up: 2, down: 0, medianPct: '+2.1', sample: 2 }],
        },
      },
      isLoading: false,
      isError: false,
    }),
  };
});

vi.mock('@/components/molecules/SymbolSuggest', () => ({
  SymbolSuggest: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <input aria-label="Symbol" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

describe('ForensicsPage', () => {
  it('renders title, phases, and similar matches', async () => {
    renderWithProviders(<ForensicsPage />, {
      routerEntries: ['/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z'],
    });
    expect(await screen.findByText(/market forensics|piyasa incelemesi/i)).toBeInTheDocument();
    expect(screen.getByText(/why this may have happened|neden olmuş olabilir/i)).toBeInTheDocument();
    expect(screen.getByText(/less support under the market/i)).toBeInTheDocument();
    expect(screen.getByText(/similar past moves|benzer geçmiş hareketler/i)).toBeInTheDocument();
    expect(screen.getByText('+3.5')).toBeInTheDocument();
  });

  it('renders the now scan of similar current setups', async () => {
    renderWithProviders(<ForensicsPage />, {
      routerEntries: ['/forensics?view=now'],
    });
    expect(await screen.findByText(/1 of 4 scanned/i)).toBeInTheDocument();
    expect(screen.getByText('ETHUSDT')).toBeInTheDocument();
    expect(screen.getByText(/kept rising/i)).toBeInTheDocument();
  });
});
