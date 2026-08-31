import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { LiquidationHunt } from './LiquidationHunt';
import type { HuntReport } from './LiquidationHunt.types';

vi.mock('@/libs/hooks', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/hooks')>();
  return {
    ...actual,
    useDisplayCurrency: () => ({
      formatCompact: (v: string | number | null | undefined) => String(v ?? '—'),
      formatPrice: (v: string | number | null | undefined) => String(v ?? '—'),
    }),
  };
});

const sample: HuntReport = {
  symbol: 'BTCUSDT',
  bias: {
    lean: 'up',
    margin: 20,
    upScore: 68,
    downScore: 41,
    summary: 'Up looks easier (68 vs 41). Shorts are crowded.',
    coverage: { score: 88, level: 'complete', usable: true, summary: 'Inputs look complete.' },
  },
  coverage: { score: 88, level: 'complete', usable: true, summary: 'Inputs look complete.' },
  venues: [
    {
      exchange: 'binance',
      price: '64000',
      openInterestValue: '100000000',
      fundingPayer: 'short',
      coverage: {
        score: 88,
        level: 'complete',
        usable: true,
        summary: 'Inputs look complete.',
        inputs: [
          { id: 'book', label: 'Spot book', status: 'ok' },
          { id: 'flow', label: 'Taker + recent liqs', status: 'weak', have: '18m', need: '1h', coverPct: 30 },
          { id: 'oi', label: 'Open interest', status: 'weak', age: '2h', need: '1h', coverPct: 50, stale: true },
        ],
      },
      upScore: {
        direction: 'up',
        score: 68,
        level: 'likely',
        reasons: ['Shorts are crowded'],
        factors: [
          { id: 'crowding', label: 'Crowding + funding', score: 80, sharePct: 22, effect: 6.6 },
          { id: 'proximity', label: 'Distance to zone', score: 70, sharePct: 20, effect: 4.0 },
        ],
      },
      downScore: { direction: 'down', score: 41, level: 'mixed', reasons: ['Target is farther'] },
      upHunt: {
        target: { price: '64800', movePct: '1.25' },
        spot: { notional: '1200000', reachable: true },
        estLiquidated: '8000000',
        efficiency: '6.6',
        netWithCascade: '12000',
        houseEdge: 'profit',
      },
      downHunt: {
        target: { price: '62800', movePct: '-1.88' },
        spot: { notional: '3400000', reachable: true },
        estLiquidated: '5100000',
        efficiency: '1.5',
        netWithCascade: '-40000',
        houseEdge: 'loss',
      },
    },
  ],
};

describe('LiquidationHunt', () => {
  it('compares up and down with scores and reasons', async () => {
    renderWithTheme(<LiquidationHunt data={sample} />);
    expect(await screen.findByTestId('liquidation-hunt')).toBeInTheDocument();
    expect(screen.getByText(/up looks easier/i)).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-up')).toHaveTextContent('68');
    expect(screen.getByTestId('liquidation-hunt-down')).toHaveTextContent('41');
    expect(screen.getAllByText(/shorts are crowded/i).length).toBeGreaterThan(0);
    expect(screen.getByTestId('liquidation-hunt-up')).toHaveTextContent('1200000');
    expect(screen.getByTestId('liquidation-hunt-down')).toHaveTextContent('3400000');
    expect(screen.getAllByTestId('liquidation-hunt-coverage').length).toBeGreaterThan(0);
    expect(screen.getAllByText(/inputs look complete/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/18m \/ 1h/i)).toBeInTheDocument();
    expect(screen.getByText(/2h old \/ 1h/i)).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-up-factors')).toHaveTextContent('+6.6');
    expect(screen.getByTestId('liquidation-hunt-up-factors')).toHaveTextContent('Crowding + funding');
  });

  it('marks an unusable venue as excluded', async () => {
    renderWithTheme(
      <LiquidationHunt
        data={{
          ...sample,
          bias: {
            ...sample.bias,
            summary: 'Combined uses binance only. Excluded from combined: bybit.',
            excluded: ['bybit'],
          },
          venues: [
            ...(sample.venues ?? []),
            {
              exchange: 'bybit',
              error: 'book: timeout',
              coverage: {
                score: 22,
                level: 'insufficient',
                usable: false,
                summary: 'Insufficient data (Spot book missing).',
                missing: ['Spot book'],
                inputs: [{ id: 'book', label: 'Spot book', status: 'error', detail: 'timeout' }],
              },
            },
          ],
        }}
      />,
    );
    expect(await screen.findByText(/not used in combined/i)).toBeInTheDocument();
    expect(screen.getByText(/book: timeout/i)).toBeInTheDocument();
  });
});
