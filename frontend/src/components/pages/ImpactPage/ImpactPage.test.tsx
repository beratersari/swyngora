import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { ImpactPage } from './ImpactPage';

const load = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanMarketImpactQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: 'A 100000 USDT buy slips 0.20% at the median.',
  hits: [
    { symbol: 'ETHUSDT', scope: 'combined', slippagePct: 1.4, slippageVsBestPct: 1.1, impactPct: 0.8, impactAvailable: true, exhausted: true },
    { symbol: 'BTCUSDT', scope: 'combined', slippagePct: 0.05, slippageVsBestPct: 0.02, impactPct: 0, impactAvailable: true, exhausted: false },
  ],
};

describe('ImpactPage', () => {
  beforeEach(() => {
    load.mockReset();
    refetch.mockReset();
  });

  it('ranks the wider book first for the same order', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<ImpactPage />, { routerEntries: ['/impact'] });
    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    const symbols = await screen.findAllByText(/BTC\/USDT|ETH\/USDT/);
    expect(symbols[0]).toHaveTextContent('ETH/USDT');
    expect(screen.getByText('1.40%')).toBeInTheDocument();
    expect(screen.getByText(/0\.20%/)).toBeInTheDocument();
  });
});
