import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { OrderReasonNote } from './OrderReasonNote';

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      isSuccess: true,
      data: { priceCvd: { kind: 'agrees' } },
    }),
  };
});

describe('OrderReasonNote', () => {
  it('names the clause the live read left', () => {
    renderWithTheme(
      <OrderReasonNote
        symbol="BTCUSDT"
        entry={{ priceCvd: { kind: 'price_up_cvd_down' } }}
      />,
    );
    expect(screen.getByText(/left the reason on this order/i)).toBeInTheDocument();
    expect(screen.getByText(/price is rising while futures cvd is falling/i)).toBeInTheDocument();
  });
});
