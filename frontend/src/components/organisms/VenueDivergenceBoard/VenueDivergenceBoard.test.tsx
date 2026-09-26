import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { VenueDivergenceBoard } from './VenueDivergenceBoard';

describe('VenueDivergenceBoard', () => {
  it('shows OKX as the venue moving differently', () => {
    renderWithProviders(
      <VenueDivergenceBoard
        report={{
          summary: 'OKX is moving differently',
          groups: [
            { lean: 'bullish', exchanges: ['binance', 'bybit'] },
            { lean: 'bearish', exchanges: ['okx'] },
          ],
          outliers: ['okx'],
          diffs: [
            {
              metric: 'positioning',
              label: 'Price + OI regime',
              readings: [
                { exchange: 'binance', value: 'long buildup' },
                { exchange: 'bybit', value: 'long buildup' },
                { exchange: 'okx', value: 'short buildup' },
              ],
            },
          ],
        }}
      />,
    );
    expect(screen.getByText(/binance, bybit \(bullish\)/)).toBeInTheDocument();
    expect(screen.getByText(/okx \(bearish\)/)).toBeInTheDocument();
    expect(screen.getByText(/Moving differently: okx/)).toBeInTheDocument();
    expect(screen.getByText(/okx short buildup/)).toBeInTheDocument();
  });
});
