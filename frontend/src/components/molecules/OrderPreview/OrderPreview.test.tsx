import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import type { OrderPreview as OrderPreviewModel } from '@/libs/api';
import { renderWithTheme } from '@/test/render';
import { OrderPreview } from './OrderPreview';

const blockedBuy: OrderPreviewModel = {
  product: 'spot',
  type: 'market',
  side: 'buy',
  symbol: 'BTCUSDT',
  fillPrice: 100,
  fee: 1,
  cashDebit: 101,
  cashAfter: 899,
  availableAfter: 899,
  fillsNow: true,
  risk: {
    applies: true,
    blocked: true,
    reasons: ['BTC would be 40.00% of the portfolio (limit 30.00%)'],
    asset: 'BTC',
    weightLimitHit: true,
    projectedWeightPct: 40,
  },
  legs: [],
};

describe('OrderPreview', () => {
  it('shows the fee and that risk limits would block the buy', () => {
    renderWithTheme(<OrderPreview preview={blockedBuy} />);
    expect(screen.getByText(/^fee$|^komisyon$/i)).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
    expect(
      screen.getByText(/risk limits would block this order|risk limitleri bu emri durdururdu/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/over its weight limit|ağırlık limitini aşardı/i)).toBeInTheDocument();
    expect(screen.queryByText(/40\.00% of the portfolio/i)).not.toBeInTheDocument();
  });

  it('shows the size that loses the risk budget at the stop', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          sizing: {
            riskPercent: 1,
            riskBudget: 100,
            estimatedLoss: 100,
            quantity: 10,
            stopAttached: false,
            stopBeyondLiquidation: false,
          },
        }}
      />,
    );
    expect(screen.getByText(/1% of the book|defterin 1%/i)).toBeInTheDocument();
    expect(screen.getByText(/does not place it|onu göndermez/i)).toBeInTheDocument();
    expect(screen.getByText(/risk budget|risk bütçesi/i)).toBeInTheDocument();
  });

  it('says the visible book cannot hold the size', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          fillsNow: false,
          bookSource: 'book',
          bookExhausted: true,
          bookVisibleQuantity: 1,
          fillPrice: 100,
        }}
      />,
    );
    expect(screen.getByText(/average of the visible book|görünen defterin ortalaması/i)).toBeInTheDocument();
    expect(
      screen.getByText(/cannot hold this size|bu büyüklüğü taşımıyor/i),
    ).toBeInTheDocument();
  });

  it('lists each venue and the cash saved against one exchange', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          routed: true,
          side: 'buy',
          fillsNow: true,
          risk: { ...blockedBuy.risk, blocked: false, weightLimitHit: false },
          routeLegs: [
            { exchange: 'binance', symbol: 'BTCUSDT', quantity: 0.4, fillPrice: 100, fee: 0.04 },
            { exchange: 'bybit', symbol: 'BTCUSDT', quantity: 0.6, fillPrice: 100.05, fee: 0.06 },
          ],
          missingVenues: ['coinbase'],
          compare: { exchange: 'binance', known: true, savedCash: 0.57 },
        }}
      />,
    );
    expect(screen.getAllByTestId('route-leg')).toHaveLength(2);
    expect(screen.getByTestId('route-saved')).toHaveTextContent(/0[.,]57/);
    expect(screen.getByText(/left out|dışarıda/i)).toBeInTheDocument();
    expect(screen.getByText(/after the taker fee|komisyon sonrası/i)).toBeInTheDocument();
  });

  it('shows a 10-slice plan and the first slice', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          routed: true,
          side: 'buy',
          spread: true,
          fillsNow: false,
          sliceCount: 15,
          spreadMinutes: 30,
          sliceIntervalSec: 129,
          sliceQuantity: 0.0667,
          risk: { ...blockedBuy.risk, blocked: false, weightLimitHit: false },
          routeLegs: [
            { exchange: 'binance', symbol: 'BTCUSDT', quantity: 0.1, fillPrice: 100, fee: 0.01 },
          ],
          compare: { exchange: 'coinbase', known: true, savedCash: 0.2 },
        }}
      />,
    );
    expect(screen.getByTestId('spread-plan')).toHaveTextContent(/15/);
    expect(screen.getByTestId('spread-plan')).toHaveTextContent(/30/);
    expect(screen.getByTestId('spread-plan')).toHaveTextContent(/2 min 9 s|2 dk 9 sn/);
    expect(screen.getByTestId('route-leg')).toHaveTextContent(/binance/i);
    expect(screen.getByTestId('route-saved')).toHaveTextContent(/0[.,]2/);
    expect(screen.getByText(/later slices choose again|sonraki dilimler yeniden/i)).toBeInTheDocument();
  });

  it('says a best-cost order past the slippage limit would not fill', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          routed: true,
          side: 'buy',
          fillsNow: false,
          risk: { ...blockedBuy.risk, blocked: false, weightLimitHit: false },
          maxSlippagePct: 0.5,
          slippageLimitPrice: 100.5,
          slippageLimited: true,
          slippageInsideQuantity: 0.4,
        }}
      />,
    );
    expect(screen.getByTestId('route-slippage')).toHaveTextContent(/0[.,]5%/);
    expect(screen.getByText(/inside the slippage limit|kayma limiti içinde/i)).toBeInTheDocument();
  });

  it('says nothing was placed', () => {
    renderWithTheme(<OrderPreview preview={blockedBuy} />);
    expect(screen.getByText(/preview only|yalnızca önizleme/i)).toBeInTheDocument();
  });

  it('shows when the order raises the day swing', () => {
    renderWithTheme(
      <OrderPreview
        preview={{
          ...blockedBuy,
          correlation: {
            window: '24h',
            interval: '5m',
            gross: 200,
            samples: 30,
            swingPct: 2,
            apartPct: 1,
            holdings: [],
            pairs: [],
            missing: [],
            note: 'note',
            add: { asset: 'SOL', notional: 100, side: 'long', swingPct: 4.5, samples: 30, complete: true },
          },
        }}
      />,
    );
    expect(screen.getByTestId('preview-correlation')).toHaveTextContent(/2% to 4\.5%/i);
  });
});
