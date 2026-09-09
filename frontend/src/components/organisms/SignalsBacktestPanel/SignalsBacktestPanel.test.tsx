import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SignalsBacktestPanel } from './SignalsBacktestPanel';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

describe('SignalsBacktestPanel', () => {
  it('renders start controls', () => {
    renderWithProviders(
      <SignalsBacktestPanel
        rules={[{ id: 'r1', type: 'rsi', interval: '4h', enabled: true, rsiThreshold: 40 }]}
        profiles={[]}
        jobs={[]}
        signals={[]}
        rangeOptions={[{ value: '90d', label: '90d' }]}
        onStart={() => undefined}
        onSelect={() => undefined}
        onCancel={() => undefined}
      />,
    );
    expect(screen.getByRole('button', { name: /run backtest|geriye dönük/i })).toBeInTheDocument();
  });

  it('shows first-touch trade result in hit details and performance', () => {
    renderWithProviders(
      <SignalsBacktestPanel
        rules={[{ id: 'r1', type: 'rsi', interval: '4h', enabled: true, rsiThreshold: 40 }]}
        profiles={[]}
        jobs={[
          {
            id: 'job-1',
            ruleId: 'r1',
            kind: 'custom_profile',
            exchange: 'binance',
            symbol: 'ETHUSDT',
            interval: '4h',
            rangeStart: '2024-01-01T00:00:00Z',
            rangeEnd: '2024-04-01T00:00:00Z',
            status: 'completed',
            progressPct: 100,
            signalCount: 1,
          },
        ]}
        signals={[
          {
            id: 'sig-1',
            backtestId: 'job-1',
            signalAt: '2024-02-01T00:00:00Z',
            closePrice: 100,
            summary: 'Custom mix B (70) crossed 62',
            trade: {
              side: 'long',
              leverage: 4,
              entry: 100,
              stopLoss: 97,
              takeProfit: 105,
              outcome: 'tp',
              pnlPct: 20,
              barsHeld: 3,
            },
          },
        ]}
        performance={{
          signalCount: 1,
          return1d: { days: 1, sampleCount: 1, average: 2, median: 2, winRate: 100 },
          return5d: { days: 5, sampleCount: 1, average: 4, median: 4, winRate: 100 },
          return20d: { days: 20, sampleCount: 0 },
          trades: { sampleCount: 1, tpHits: 1, slHits: 0, openCount: 0, longCount: 1, shortCount: 0, winRate: 100, average: 20, median: 20 },
        }}
        rangeOptions={[{ value: '90d', label: '90d' }]}
        selectedId="job-1"
        selectedJob={{
          id: 'job-1',
          ruleId: 'r1',
          kind: 'custom_profile',
          exchange: 'binance',
          symbol: 'ETHUSDT',
          interval: '4h',
          rangeStart: '2024-01-01T00:00:00Z',
          rangeEnd: '2024-04-01T00:00:00Z',
          status: 'completed',
          progressPct: 100,
          signalCount: 1,
        }}
        onStart={() => undefined}
        onSelect={() => undefined}
        onCancel={() => undefined}
      />,
    );
    expect(screen.getByText(/take profit first|önce kâr al/i)).toBeInTheDocument();
    expect(screen.getByText(/trade results|işlem sonuçları/i)).toBeInTheDocument();
  });
});
