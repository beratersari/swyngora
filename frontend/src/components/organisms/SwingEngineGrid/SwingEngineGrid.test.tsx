import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SwingEngineGrid } from './SwingEngineGrid';

describe('SwingEngineGrid', () => {
  it('renders accepted setup with levels', () => {
    renderWithProviders(
      <SwingEngineGrid
        items={[
          {
            exchange: 'binance',
            symbol: 'ETHUSDT',
            interval: '4h',
            accepted: true,
            stage: 'trigger',
            setupType: 'trend_pullback',
            swingScore: 72,
            grade: 'B',
            fresh: true,
            btcRegime: 'bull',
            rsi: 42,
            barTime: '2026-08-01T04:00:00.000Z',
            levels: { entry: 100, stopLoss: 95, takeProfit: 110, rr: 2, riskPct: 5 },
          },
        ]}
      />,
    );
    expect(screen.getByText(/ETH\/USDT|ETHUSDT/)).toBeInTheDocument();
    expect(screen.getByText('trigger')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
    expect(screen.getByText('95')).toBeInTheDocument();
    expect(screen.getByText('110')).toBeInTheDocument();
    expect(screen.getByText(/04:00:00/)).toBeInTheDocument();
  });
});
