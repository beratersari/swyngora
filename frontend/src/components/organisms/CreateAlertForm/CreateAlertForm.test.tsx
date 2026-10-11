import { describe, expect, it, vi } from 'vitest';
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
    expect(screen.getByText(/leaves the read stored on your newest open buy/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/target price/i)).not.toBeInTheDocument();
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
