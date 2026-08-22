import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PostDelistPanel } from './PostDelistPanel';

describe('PostDelistPanel', () => {
  it('renders off-venue last price without a second chart', () => {
    renderWithTheme(
      <PostDelistPanel
        view={{
          available: true,
          source: 'coingecko',
          sourceLabel: 'CoinGecko',
          note: 'Not this exchange.',
          lastPrice: '0.1234',
          priceChangePercent: '-3.5',
          quote: 'USD',
          interval: '1d',
        }}
        lastPrice="0.1234"
      />,
    );
    expect(screen.getByTestId('post-delist-panel')).toBeInTheDocument();
    expect(screen.getAllByText('CoinGecko').length).toBeGreaterThan(0);
    expect(screen.getByText('0.1234')).toBeInTheDocument();
    expect(screen.getByText('-3.50%')).toBeInTheDocument();
    expect(screen.queryByText('Off-venue movement')).not.toBeInTheDocument();
  });

  it('explains when no off-venue price exists', () => {
    renderWithTheme(
      <PostDelistPanel
        view={{
          available: false,
          note: 'No public off-venue price was found after delist.',
        }}
      />,
    );
    expect(screen.getByText('No off-venue price')).toBeInTheDocument();
    expect(
      screen.getByText('No public off-venue price was found after delist.'),
    ).toBeInTheDocument();
  });
});
