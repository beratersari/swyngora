import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SpreadsPage } from './SpreadsPage';

const load = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanWideSpreadsQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: '1 books are at least 1.5× their own typical spread.',
  hits: [
    { symbol: 'BTCUSDT', exchange: 'binance', currentPct: '0.08', typicalPct: '0.01', ratio: '8' },
  ],
};

describe('SpreadsPage', () => {
  beforeEach(() => {
    load.mockReset();
  });

  it('lists a book that opened versus its own typical', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch: vi.fn(),
    }));
    renderWithProviders(<SpreadsPage />, { routerEntries: ['/spreads'] });
    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByText('8×')).toBeInTheDocument();
    expect(screen.getByText(payload.summary)).toBeInTheDocument();
  });
});
