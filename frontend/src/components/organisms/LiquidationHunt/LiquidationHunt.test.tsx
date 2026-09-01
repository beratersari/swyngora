import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
  scoreMix: { source: 'default', requestedTotal: 100, usedTotal: 100, note: 'Default weights.' },
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
          { id: 'proximity', label: 'Distance to zone', score: 70, sharePct: 20, effect: 4.0, status: 'used' },
          { id: 'book', label: 'Spot walk cost', score: 60, sharePct: 16, effect: 1.6, status: 'used' },
          { id: 'efficiency', label: 'Liq per spot', score: 55, sharePct: 12, effect: 0.6, status: 'used' },
          { id: 'trend', label: 'Price + OI trend', score: 50, sharePct: 20, effect: 0, status: 'missing' },
          { id: 'crowding', label: 'Crowding + funding', score: 80, sharePct: 18, effect: 5.4, status: 'used' },
          { id: 'flow', label: 'Taker + recent liqs', score: 58, sharePct: 14, effect: 1.1, status: 'used' },
        ],
      },
      downScore: {
        direction: 'down',
        score: 41,
        level: 'mixed',
        reasons: ['Target is farther'],
        factors: [
          { id: 'proximity', score: 40, status: 'used' },
          { id: 'book', score: 45, status: 'used' },
          { id: 'efficiency', score: 42, status: 'used' },
          { id: 'trend', status: 'missing' },
          { id: 'crowding', score: 30, status: 'used' },
          { id: 'flow', score: 44, status: 'used' },
        ],
      },
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
      upCascade: {
        direction: 'up',
        summary: 'Cascade feeds itself through zone 1. Stalls at zone 2 — extra spot is needed.',
        stallsAtIndex: 2,
        stallNote: 'Prior assumed exit flow covers 56% of this hop; desk still needs 40000.',
        chainEasier: true,
        steps: [
          {
            index: 1,
            role: 'start',
            zoneEst: 'model',
            band: { price: '64256', leverage: '125', movePct: '+0.40' },
            zoneNotional: '800000',
            standalone: { notional: '180000', reachable: true },
            reachable: true,
            note: 'First zone from last price.',
          },
          {
            index: 2,
            role: 'helped',
            zoneEst: 'model',
            band: { price: '64384', leverage: '100', movePct: '+0.60' },
            hopPct: '+0.20',
            zoneNotional: '2100000',
            incremental: { notional: '90000', reachable: true },
            remaining: { notional: '40000', reachable: true },
            easier: true,
            assistancePct: '56',
            strength: '56',
            note: 'Prior assumed exit flow covers 56% of this hop.',
          },
        ],
      },
      downCascade: {
        direction: 'down',
        summary: '2 long liquidations below zones.',
        steps: [
          {
            index: 1,
            role: 'start',
            zoneEst: 'model',
            band: { price: '63744', leverage: '125', movePct: '-0.40' },
            zoneNotional: '700000',
            standalone: { notional: '220000', reachable: true },
            reachable: true,
          },
        ],
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
    expect(screen.getByTestId('liquidation-hunt-up-factors')).toHaveTextContent('+5.4');
    expect(screen.getByTestId('liquidation-hunt-up-factors')).toHaveTextContent('Crowding + funding');
    expect(screen.queryByTestId('liquidation-hunt-path-up')).not.toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-weights')).toBeInTheDocument();
  });

  it('previews default vs draft scores in the custom weights dialog', async () => {
    const user = userEvent.setup();
    renderWithTheme(<LiquidationHunt data={sample} />);
    await user.click(screen.getByTestId('liquidation-hunt-weights'));
    expect(await screen.findByTestId('liquidation-hunt-weights-preview')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-weights-preview-default')).toHaveTextContent(/up/i);
    expect(screen.getByTestId('liquidation-hunt-weights-preview-custom')).toHaveTextContent(/up/i);
    expect(screen.getByTestId('liquidation-hunt-weights-preview-factors-up')).toHaveTextContent(/no data/i);
    expect(screen.getByTestId('liquidation-hunt-weights-preview-factors-down')).toBeInTheDocument();
  });

  it('opens the cascade path subview without the compare scores', async () => {
    renderWithTheme(<LiquidationHunt data={sample} panel="path" />);
    expect(await screen.findByTestId('liquidation-hunt-path-up')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-path-down')).toBeInTheDocument();
    expect(screen.queryByTestId('liquidation-hunt-up')).not.toBeInTheDocument();
    expect(screen.getAllByTestId('liquidation-hunt-path-step').length).toBe(3);
    expect(screen.getByTestId('liquidation-hunt-path-up-stall')).toHaveTextContent(/stops feeding itself at zone 2/i);
    expect(screen.getAllByText(/56% of this hop/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/first zone from last price/i)).toBeInTheDocument();
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
