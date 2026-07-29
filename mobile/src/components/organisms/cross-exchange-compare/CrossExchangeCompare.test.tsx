import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CrossExchangeCompare } from './CrossExchangeCompare';
import type { CrossExchangeRowModel } from '@/libs/utils';

const rows: CrossExchangeRowModel[] = [
  {
    id: 'binance|BTCUSDT',
    exchange: 'binance',
    symbol: 'BTCUSDT',
    isSource: true,
    lastPriceLabel: '67,000',
    changePercentLabel: '+1.00%',
    changeTone: 'success',
    quoteVolumeLabel: '$1.2B',
    status: 'ok',
  },
  {
    id: 'coinbase|BTC-USD',
    exchange: 'coinbase',
    symbol: 'BTC-USD',
    isSource: false,
    lastPriceLabel: '66,900',
    changePercentLabel: '+0.90%',
    changeTone: 'success',
    quoteVolumeLabel: '$800M',
    status: 'ok',
  },
];

describe('CrossExchangeCompare', () => {
  it('renders rows and navigates non-source press', () => {
    const onPress = vi.fn();
    render(
      <CrossExchangeCompare
        title="Across exchanges"
        rows={rows}
        cheapestId="coinbase|BTC-USD"
        onPressRow={onPress}
      />,
    );
    expect(screen.getByText('Across exchanges')).toBeTruthy();
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    fireEvent.click(screen.getByLabelText('coinbase BTC-USD'));
    expect(onPress).toHaveBeenCalledWith('coinbase', 'BTC-USD');
  });

  it('shows unavailable state', () => {
    render(
      <CrossExchangeCompare
        title="Across exchanges"
        rows={[
          {
            ...rows[1]!,
            status: 'unavailable',
            lastPriceLabel: '—',
          },
        ]}
        unavailableLabel="Not listed"
      />,
    );
    expect(screen.getByText('Not listed')).toBeTruthy();
  });
});
