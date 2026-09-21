import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { VolumeSurgeTable } from './VolumeSurgeTable';

describe('VolumeSurgeTable', () => {
  it('renders rows and opens on click', async () => {
    const onRowOpen = vi.fn();
    renderWithProviders(
      <VolumeSurgeTable
        hasScanned
        rows={[
          {
            exchange: 'binance',
            symbol: 'ETHUSDT',
            maxRatio: '4.2',
            hottest: '15m',
            grade: 'high',
            dominant: 'buy',
            summary: '15m is 4.2× typical',
          },
        ]}
        emptyHint="hint"
        emptyTitle="empty"
        columns={{
          symbol: 'Pair',
          maxRatio: 'Max',
          hottest: 'Hottest',
          grade: 'Grade',
          dominant: 'Side',
          summary: 'Read',
        }}
        onRowOpen={onRowOpen}
      />,
    );
    expect(screen.getByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByText('4.2×')).toBeInTheDocument();
    await userEvent.click(screen.getByText('ETH/USDT'));
    expect(onRowOpen).toHaveBeenCalledWith('binance', 'ETHUSDT');
  });
});
