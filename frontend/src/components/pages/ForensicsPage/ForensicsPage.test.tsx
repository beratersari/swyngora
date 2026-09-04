import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsPage } from './ForensicsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketForensicsQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        summary: 'Price rose during the move. Two similar cases went up after.',
        note: 'Informational only',
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
        summary: 'Price rose during the move. Two similar cases went up after.',
        note: 'Informational only',
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
    expect(screen.getByText(/price rose during the move/i)).toBeInTheDocument();
    expect(screen.getByText(/similar past moves|benzer geçmiş hareketler/i)).toBeInTheDocument();
    expect(screen.getByText('+3.5')).toBeInTheDocument();
  });
});
