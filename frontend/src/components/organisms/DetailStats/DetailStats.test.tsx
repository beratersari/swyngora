import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { DetailStats } from './DetailStats';

describe('DetailStats', () => {
  it('renders ticker stats', () => {
    renderWithTheme(
      <DetailStats
        exchange="binance"
        ticker={{
          lastPrice: '100',
          highPrice: '110',
          lowPrice: '90',
          volume: '1000',
          quoteVolume: '100000',
          priceChangePercent: '2',
          tradeCount: 5,
        }}
      />,
    );
    // Multiple formatted numbers may appear; ensure section renders
    expect(screen.getAllByText(/100|110|90/).length).toBeGreaterThan(0);
  });

  it('shows ticker and supply errors', () => {
    renderWithTheme(
      <DetailStats
        exchange="binance"
        tickerError="ticker failed"
        supplyError="supply failed"
      />,
    );
    expect(screen.getByText('ticker failed')).toBeInTheDocument();
    expect(screen.getByText('supply failed')).toBeInTheDocument();
  });

  it('shows skeletons when loading', () => {
    const { container } = renderWithTheme(
      <DetailStats exchange="binance" isLoading />,
    );
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });
});
