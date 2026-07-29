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
    showPumps: true,
    onTogglePumps: vi.fn(),
    showPumpMargin: false,
    onTogglePumpMargin: vi.fn(),
    // empty chart series avoids lightweight-charts canvas in jsdom
    candles: [],
    candleOverlays: [],
    chartMarkers: [],
    chartPriceLines: [],
    candlesLoading: false,
    candlesLoadingOlder: false,
    candlesError: null,
    chartSeriesKey: 'binance|BTCUSDT|1h',
    canLoadOlderHistory: false,
    historyEdgeBars: 15,
    onRequestOlderHistory: vi.fn(),
    rsiPoints: [],
    latestRsi: null,
    indicatorsLoading: false,
    indicatorsError: null,
    emaLatestLabels: [],
    pumpEventRows: [],
    pumpEventsLoading: false,
    pumpEventsError: null,
    pumpEventsSubtitle: null,
    pumpDisclaimer: 'Informational only — not financial advice.',
    crossExchangeTitle: 'Across exchanges',
    crossExchangeRows: [
      {
        id: 'binance|BTCUSDT',
        exchange: 'binance',
        symbol: 'BTCUSDT',
        isSource: true,
        lastPriceLabel: '67,000',
        changePercentLabel: '+1.50%',
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
        changePercentLabel: '+1.40%',
        changeTone: 'success',
        quoteVolumeLabel: '$800M',
        status: 'ok',
      },
    ],
    crossExchangeDisclaimer: 'Venue-local prices — informational only.',
    crossExchangeUnavailableLabel: 'Not listed',
    crossExchangeSourceLabel: 'This venue',
    crossExchangeCheapestLabel: 'Lowest',
    crossExchangeCheapestId: 'coinbase|BTC-USD',
    onPressCrossExchangeRow: vi.fn(),
    onBack: vi.fn(),
    onRetry: vi.fn(),
    askAiLabel: 'Ask AI about this pair',
    onAskAi: vi.fn(),
    ...overrides,
  };
}

describe('CoinDetailPage', () => {
  it('renders header stats and back', () => {
    const onBack = vi.fn();
    render(<CoinDetailPage viewModel={makeVm({ onBack })} />);
    expect(screen.getAllByText('BTCUSDT').length).toBeGreaterThan(0);
    expect(screen.getAllByText('67,000').length).toBeGreaterThan(0);
    expect(screen.getByText('Open')).toBeTruthy();
    fireEvent.click(screen.getByLabelText('Back'));
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

  it('renders cross-exchange section', () => {
    const onPress = vi.fn();
    render(
      <CoinDetailPage
        viewModel={makeVm({ onPressCrossExchangeRow: onPress })}
      />,
    );
    expect(screen.getByText('Across exchanges')).toBeTruthy();
    expect(screen.getByText('BTC-USD')).toBeTruthy();
    fireEvent.click(screen.getByLabelText('coinbase BTC-USD'));
    expect(onPress).toHaveBeenCalledWith('coinbase', 'BTC-USD');
  });
});
