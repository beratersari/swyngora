import { describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CreateAlertForm } from './CreateAlertForm';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

describe('CreateAlertForm', () => {
  it('renders create controls', () => {
    renderWithProviders(
      <CreateAlertForm defaultSymbol="BTCUSDT" onSubmit={async () => undefined} />,
    );
    expect(screen.getByRole('button', { name: /create|oluştur/i })).toBeInTheDocument();
  });

  it('explains that an entry-read alert follows the open paper buy', () => {
    renderWithProviders(
      <CreateAlertForm
        defaultKind="situation"
        defaultExchange="binance"
        defaultSymbol="BTCUSDT"
        onSubmit={async () => undefined}
      />,
    );
    expect(screen.getByText(/newest open buy filled/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/target price/i)).not.toBeInTheDocument();
  });

  it('creates a selected-read alert without an open trade', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderWithProviders(
      <CreateAlertForm
        defaultKind="situation"
        defaultCondition="matches"
        defaultExchange="binance"
        defaultSymbol="BTCUSDT"
        defaultMatch={{ priceCvd: 'price_up_cvd_down', funding: 'long' }}
        onSubmit={onSubmit}
      />,
    );
    expect(screen.getByText(/No open trade is required/i)).toBeInTheDocument();
    expect(screen.queryByText(/newest open buy filled/i)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /create alert/i }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: 'situation',
        condition: 'matches',
        symbol: 'BTCUSDT',
        situation: { priceCvd: 'price_up_cvd_down', funding: 'long' },
      }),
    );
  });

  it('keeps a selected read without a clause from being created', () => {
    renderWithProviders(
      <CreateAlertForm
        defaultKind="situation"
        defaultCondition="matches"
        defaultExchange="binance"
        defaultSymbol="BTCUSDT"
        onSubmit={async () => undefined}
      />,
    );
    expect(screen.getByRole('button', { name: /create alert/i })).toBeDisabled();
  });

  it('explains that all-exchange funding watches every venue', () => {
    renderWithProviders(
      <CreateAlertForm
        defaultKind="funding"
        defaultExchange="all"
        defaultSymbol="BTCUSDT"
        onSubmit={async () => undefined}
      />,
    );
    expect(screen.getByText(/any of Binance, Bybit, or OKX|Binance, Bybit veya OKX/i)).toBeInTheDocument();
  });
});
