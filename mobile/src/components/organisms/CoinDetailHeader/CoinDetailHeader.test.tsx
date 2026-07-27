import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CoinDetailHeader } from './CoinDetailHeader';

describe('CoinDetailHeader', () => {
  it('renders symbol and back', () => {
    const onBack = vi.fn();
    render(
      <CoinDetailHeader
        symbol="BTCUSDT"
        exchange="binance"
        lastPriceLabel="67,000"
        changePercentLabel="+1%"
        changeTone="success"
        onBack={onBack}
      />,
    );
    expect(screen.getByText('BTCUSDT')).toBeTruthy();
    fireEvent.click(screen.getByText('← Back'));
    expect(onBack).toHaveBeenCalled();
  });

  it('renders star when provided', () => {
    const onStar = vi.fn();
    render(
      <CoinDetailHeader
        symbol="BTCUSDT"
        exchange="binance"
        lastPriceLabel="67,000"
        changePercentLabel="+1%"
        changeTone="success"
        onBack={vi.fn()}
        watched={false}
        onStarPress={onStar}
      />,
    );
    fireEvent.click(screen.getByLabelText('Add BTCUSDT to favorites'));
    expect(onStar).toHaveBeenCalled();
  });
});
