import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { WatchlistList } from './WatchlistList';

const rows = [
  {
    id: 'binance|BTCUSDT',
    exchange: 'binance',
    symbol: 'BTCUSDT',
    lastPriceLabel: '1',
    changePercentLabel: '+1%',
    changeTone: 'success' as const,
  },
];

describe('WatchlistList', () => {
  it('shows empty message', () => {
    render(
      <WatchlistList
        rows={[]}
        isLoading={false}
        emptyMessage="No favorites yet"
        errorMessage={null}
        onRetry={vi.fn()}
        onPressRow={vi.fn()}
        onUnstar={vi.fn()}
      />,
    );
    expect(screen.getByText('No favorites yet')).toBeTruthy();
  });

  it('shows error and retry', () => {
    const onRetry = vi.fn();
    render(
      <WatchlistList
        rows={[]}
        isLoading={false}
        emptyMessage={null}
        errorMessage="Network error"
        onRetry={onRetry}
        onPressRow={vi.fn()}
        onUnstar={vi.fn()}
      />,
    );
    expect(screen.getByText('Network error')).toBeTruthy();
    fireEvent.click(screen.getByText('Retry'));
    expect(onRetry).toHaveBeenCalled();
  });

  it('renders rows', () => {
    render(
      <WatchlistList
        rows={rows}
        isLoading={false}
        emptyMessage={null}
        errorMessage={null}
        onRetry={vi.fn()}
        onPressRow={vi.fn()}
        onUnstar={vi.fn()}
      />,
    );
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
  });
});
