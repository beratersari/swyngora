import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { CreateAlertForm } from './CreateAlertForm';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
    useGetFxRatesQuery: () => ({
      data: { rates: { TRY: 34, EUR: 0.92, USD: 1 }, asOf: '2026-01-01' },
      isError: false,
    }),
  };
});

describe('CreateAlertForm display currency (finding 7)', () => {
  // Finding 7 was a false positive as a major bug: the form never prefills a
  // converted last price and never labels TRY/EUR. Typed numbers are native quote.
  it('posts the raw typed number with no FX conversion', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderWithProviders(
      <CreateAlertForm defaultExchange="binance" defaultSymbol="BTCUSDT" onSubmit={onSubmit} />,
    );

    const price = screen.getByRole('spinbutton', { name: /target price|eşik|threshold/i });
    await user.clear(price);
    await user.type(price, '3400000');
    await user.click(screen.getByRole('button', { name: /create|oluştur/i }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit.mock.calls[0][0]).toMatchObject({
      symbol: 'BTCUSDT',
      targetPrice: 3400000,
    });
  });

  it('does not prefill or label a converted last price', () => {
    renderWithProviders(
      <CreateAlertForm defaultExchange="binance" defaultSymbol="BTCUSDT" onSubmit={async () => undefined} />,
    );
    const price = screen.getByRole('spinbutton', { name: /target price|eşik|threshold/i });
    expect((price as HTMLInputElement).value === '' || (price as HTMLInputElement).value === '0').toBe(
      true,
    );
    expect(screen.queryByText(/TRY|EUR|USDT/i)).toBeNull();
  });
});
