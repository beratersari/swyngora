import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MarketsList } from './MarketsList';

const rows = [
  {
    id: 'BTCUSDT',
    symbol: 'BTCUSDT',
    lastPriceLabel: '1',
    changePercentLabel: '+1%',
    changeTone: 'success' as const,
    quoteVolumeLabel: '1B',
    marketCapLabel: '1T',
    tagsLabel: '',
  },
];

describe('MarketsList', () => {
  it('renders rows with stars when handlers provided', () => {
    render(
      <MarketsList
        rows={rows}
        isLoading={false}
        emptyMessage={null}
        errorMessage={null}
        onRetry={vi.fn()}
        onPressRow={vi.fn()}
        isWatched={() => true}
        onStarPress={vi.fn()}
      />,
    );
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByLabelText('Remove BTCUSDT from favorites')).toBeTruthy();
  });

  it('shows empty state', () => {
    render(
      <MarketsList
        rows={[]}
        isLoading={false}
        emptyMessage="No markets match filters"
        errorMessage={null}
        onRetry={vi.fn()}
        onPressRow={vi.fn()}
      />,
    );
    expect(screen.getByText('No markets match filters')).toBeTruthy();
  });
});
