import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { GapsPage } from './GapsPage';

const load = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanBookGapsQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: '1 coins easier up, 0 easier down, across 2 pairs',
  hits: [
    { symbol: 'SOLUSDT', scope: 'all', easier: 'up', upNotional: '8000', downNotional: '80000', upGapPct: 45, downGapPct: 4, widestPct: 0.8 },
    { symbol: 'BTCUSDT', scope: 'all', easier: 'even', upNotional: '500000', downNotional: '480000', upGapPct: 10, downGapPct: 12, widestPct: 0.2 },
  ],
};

describe('GapsPage', () => {
  beforeEach(() => {
    load.mockReset();
  });

  it('lists the thinner book first', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch: vi.fn(),
    }));
    renderWithProviders(<GapsPage />, { routerEntries: ['/gaps'] });
    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    const symbols = await screen.findAllByText(/SOL\/USDT|BTC\/USDT/);
    expect(symbols[0]).toHaveTextContent('SOL/USDT');
    expect(screen.getByText('8,000')).toBeInTheDocument();
    expect(screen.getByText('45.00%')).toBeInTheDocument();
  });
});
