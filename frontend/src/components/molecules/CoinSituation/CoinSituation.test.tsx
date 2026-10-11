import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { CoinSituation } from './CoinSituation';

describe('CoinSituation', () => {
  it('names a missing venue and keeps that clause blank of a number', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'price_up_cvd_down' },
          openInterestMissing: ['bybit', 'okx'],
          liquidation: { side: 'long', distancePct: 1.2 },
        }}
      />,
    );
    expect(screen.getByRole('region', { name: 'Situation' })).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(
      screen.getByText('Open interest stays blank until Bybit and OKX have the last hour.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Long liquidations below price are closer (1.2%).')).toBeInTheDocument();
    expect(screen.queryByText(/building/)).not.toBeInTheDocument();
  });

  it('adds funding, the crowd gap, and whale versus retail when those clauses are set', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          funding: { payer: 'long', ratePct: 0.01 },
          crowd: { lean: 'top_more_long', gapPct: 7 },
          size: { kind: 'whale_buy_retail_sell' },
          sizeMissing: [],
        }}
      />,
    );
    expect(screen.getByText('Longs pay the combined funding (0.01%).')).toBeInTheDocument();
    expect(screen.getByText('Top traders are more long than the crowd (7 pp).')).toBeInTheDocument();
    expect(screen.getByText('Large prints are buying while smaller prints are selling.')).toBeInTheDocument();
  });
});
