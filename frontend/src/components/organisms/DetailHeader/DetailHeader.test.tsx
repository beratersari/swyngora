import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { DetailHeader } from './DetailHeader';

describe('DetailHeader', () => {
  it('renders symbol and exchange', () => {
    renderWithProviders(
      <DetailHeader
        symbol="BTCUSDT"
        exchange="binance"
        lastPrice="67000"
        priceChangePercent="1.5"
        backTo="/markets"
      />,
    );
    expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByText(/binance/i)).toBeInTheDocument();
  });

  it('shows off-venue 24h percent on a halted last print', () => {
    renderWithProviders(
      <DetailHeader
        symbol="PYRUSDT"
        exchange="binance"
        lastPrice="0.021"
        priceChangePercent="-12.5"
        halted
        backTo="/markets"
      />,
    );
    expect(screen.getByText('-12.50%')).toBeInTheDocument();
  });

  it('shows loading skeletons for price when isLoading', () => {
    const { container } = renderWithProviders(
      <DetailHeader
        symbol="BTCUSDT"
        exchange="binance"
        isLoading
        backTo="/markets"
      />,
    );
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });
});
