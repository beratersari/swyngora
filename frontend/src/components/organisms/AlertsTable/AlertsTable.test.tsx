import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AlertsTable } from './AlertsTable';

describe('AlertsTable', () => {
  it('renders empty state', () => {
    renderWithProviders(
      <AlertsTable items={[]} loading={false} onDelete={vi.fn()} />,
    );
    expect(screen.getByText(/no alerts|henüz/i)).toBeInTheDocument();
  });

  it('renders a price alert row', () => {
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: '1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            condition: 'above',
            targetPrice: 100,
            mode: 'one_time',
            status: 'active',
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/BTC/)).toBeInTheDocument();
  });

  it('names the venue that crossed an all-exchange funding alert', () => {
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: '2',
            exchange: 'all',
            symbol: 'BTCUSDT',
            kind: 'funding',
            condition: 'above',
            targetPrice: 0.05,
            mode: 'one_time',
            status: 'triggered',
            triggeredPrice: 0.08,
            triggeredVenue: 'bybit',
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/bybit/i)).toBeInTheDocument();
  });
});
