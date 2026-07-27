import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { MarketsTable } from './MarketsTable';

const item = {
  symbol: 'ETHUSDT',
  baseAsset: 'ETH',
  quoteAsset: 'USDT',
  lastPrice: '3000',
  priceChangePercent: '-1.5',
  quoteVolume: '5000000',
};

describe('MarketsTable', () => {
  it('renders market rows', () => {
    renderWithTheme(
      <MarketsTable
        items={[item]}
        exchange="binance"
        sort="quoteVolume"
        order="desc"
        total={1}
        limit={50}
        offset={0}
        onSortChange={() => undefined}
        onPageChange={() => undefined}
      />,
    );
    expect(screen.getByText('ETH/USDT')).toBeInTheDocument();
  });

  it('shows loading skeleton when loading and empty', () => {
    const { container } = renderWithTheme(
      <MarketsTable
        items={[]}
        exchange="binance"
        sort="quoteVolume"
        order="desc"
        total={0}
        limit={50}
        offset={0}
        isLoading
        onSortChange={() => undefined}
        onPageChange={() => undefined}
      />,
    );
    expect(container.querySelector('.ant-skeleton') || screen.getByRole('status')).toBeTruthy();
  });

  it('shows error message when provided', () => {
    renderWithTheme(
      <MarketsTable
        items={[]}
        exchange="binance"
        sort="quoteVolume"
        order="desc"
        total={0}
        limit={50}
        offset={0}
        errorMessage="Network down"
        onSortChange={() => undefined}
        onPageChange={() => undefined}
        onRetry={vi.fn()}
      />,
    );
    expect(screen.getByText(/Network down/i)).toBeInTheDocument();
  });
});
