import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CoinDetailPage } from './CoinDetailPage';
import type { CoinDetailPageViewModel } from './CoinDetailPage.types';

function makeVm(overrides: Partial<CoinDetailPageViewModel> = {}): CoinDetailPageViewModel {
  return {
    symbol: 'BTCUSDT',
    exchange: 'binance',
    lastPriceLabel: '67,000',
    changePercentLabel: '+1.50%',
    changeTone: 'success',
    headerLoading: false,
    watched: false,
    onStarPress: vi.fn(),
    actionError: null,
    statsItems: [
      { label: 'Open', value: '66,000' },
      { label: 'High 24h', value: '68,000' },
    ],
    statsLoading: false,
    tickerError: null,
    supplyError: null,
    intervals: ['1h', '4h'],
    intervalsLoading: false,
    interval: '1h',
    onSelectInterval: vi.fn(),
    showEma: true,
    onToggleEma: vi.fn(),
    // empty chart series avoids lightweight-charts canvas in jsdom
    candles: [],
    candleOverlays: [],
    candlesLoading: false,
    candlesError: null,
    rsiPoints: [],
    latestRsi: null,
    indicatorsLoading: false,
    indicatorsError: null,
    emaLatestLabels: [],
    onBack: vi.fn(),
    onRetry: vi.fn(),
    ...overrides,
  };
}

describe('CoinDetailPage', () => {
  it('renders header stats and back', () => {
    const onBack = vi.fn();
    render(<CoinDetailPage viewModel={makeVm({ onBack })} />);
    expect(screen.getAllByText('BTCUSDT').length).toBeGreaterThan(0);
    expect(screen.getByText('67,000')).toBeTruthy();
    expect(screen.getByText('Open')).toBeTruthy();
    fireEvent.click(screen.getByText('← Back'));
    expect(onBack).toHaveBeenCalled();
  });

  it('shows candle error', () => {
    render(
      <CoinDetailPage
        viewModel={makeVm({
          candles: [],
          candlesError: 'Candle fetch failed',
        })}
      />,
    );
    expect(screen.getByText('Candle fetch failed')).toBeTruthy();
  });
});
