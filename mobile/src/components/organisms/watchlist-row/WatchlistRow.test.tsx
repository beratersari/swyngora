import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { WatchlistRow } from './WatchlistRow';

const row = {
  id: 'binance|BTCUSDT',
  exchange: 'binance',
  symbol: 'BTCUSDT',
  lastPriceLabel: '67,000',
  changePercentLabel: '+1%',
  changeTone: 'success' as const,
};

describe('WatchlistRow', () => {
  it('renders pair and exchange', () => {
    render(<WatchlistRow row={row} />);
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('binance')).toBeTruthy();
  });

  it('press opens detail callback', () => {
    const onPress = vi.fn();
    render(<WatchlistRow row={row} onPress={onPress} />);
    fireEvent.click(screen.getByText('BTCUSDT'));
    expect(onPress).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });

  it('unstar callback', () => {
    const onUnstar = vi.fn();
    render(<WatchlistRow row={row} onUnstar={onUnstar} />);
    fireEvent.click(screen.getByLabelText('Remove BTCUSDT from favorites'));
    expect(onUnstar).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });

  it('renders optional RSI badge', () => {
    render(
      <WatchlistRow row={{ ...row, rsiLabel: 'RSI 28.0', rsiTone: 'success' }} />,
    );
    expect(screen.getByText('RSI 28.0')).toBeTruthy();
  });
});
