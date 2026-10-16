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
