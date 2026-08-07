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
});
