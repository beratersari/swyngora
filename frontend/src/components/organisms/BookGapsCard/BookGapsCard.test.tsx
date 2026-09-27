import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { BookGapsCard } from './BookGapsCard';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetBookGapsQuery: () => ({
      data: {
        easier: 'up',
        upNotional: '12000',
        downNotional: '90000',
        upGapPct: 40,
        downGapPct: 5,
        summary: 'Asks are thinner.',
        gaps: [
          { direction: 'up', fromPrice: '100.2', toPrice: '100.8', widthPct: 0.6, notional: '500' },
        ],
      },
      isError: false,
      isFetching: false,
      error: undefined,
    }),
  };
});

describe('BookGapsCard', () => {
  it('shows that a move up meets less size and lists the empty stretch', () => {
    renderWithProviders(<BookGapsCard exchange="binance" symbol="SOLUSDT" />);
    expect(screen.getByText(/easier up|yukarı/i)).toBeInTheDocument();
    expect(screen.getByText(/100\.2/)).toBeInTheDocument();
    expect(screen.getByText(/12,000/)).toBeInTheDocument();
  });
});
