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
    expect(
      screen.getByText(/over its weight limit|ağırlık limitini aşardı/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/40\.00% of the portfolio/i)).not.toBeInTheDocument();
  });

  it('says nothing was placed', () => {
    renderWithTheme(<OrderPreview preview={blockedBuy} />);
    expect(screen.getByText(/preview only|yalnızca önizleme/i)).toBeInTheDocument();
  });
});
