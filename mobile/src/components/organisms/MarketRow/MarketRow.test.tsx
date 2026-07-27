import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarketRow } from './MarketRow';

const row = {
  id: 'BTCUSDT',
  symbol: 'BTCUSDT',
  lastPriceLabel: '67,000',
  changePercentLabel: '+1.50%',
  changeTone: 'success' as const,
  quoteVolumeLabel: '1.50B',
  marketCapLabel: '1.30T',
  tagsLabel: 'Layer1',
};

describe('MarketRow', () => {
  it('renders symbol and metrics', () => {
    render(<MarketRow row={row} onPress={vi.fn()} />);
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    expect(screen.getByText('67,000')).toBeTruthy();
    expect(screen.getByText(/Vol 1.50B/)).toBeTruthy();
  });

  it('fires onPress for row body', () => {
    const onPress = vi.fn();
    render(<MarketRow row={row} onPress={onPress} />);
    fireEvent.click(screen.getByText('BTCUSDT'));
    expect(onPress).toHaveBeenCalledWith('BTCUSDT');
  });

  it('star does not trigger row press', () => {
    const onPress = vi.fn();
    const onStar = vi.fn();
    render(
      <MarketRow row={row} onPress={onPress} watched={false} onStarPress={onStar} />,
    );
    fireEvent.click(screen.getByLabelText('Add BTCUSDT to favorites'));
    expect(onStar).toHaveBeenCalledWith('BTCUSDT');
    expect(onPress).not.toHaveBeenCalled();
  });
});
