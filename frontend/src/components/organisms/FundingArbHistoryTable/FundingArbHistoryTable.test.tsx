import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { defaultFundingHistoryRange } from './helpers';
import { FundingArbHistoryTable } from './FundingArbHistoryTable';

describe('FundingArbHistoryTable', () => {
  it('renders an OKX short stretch', () => {
    renderWithProviders(
      <FundingArbHistoryTable
        history={{
          summary: '1 after-fee winning stretch(es).',
          runs: [
            {
              startedAt: '2026-08-01T08:00:00Z',
              endedAt: '2026-08-01T16:00:00Z',
              longExchange: 'binance',
              shortExchange: 'okx',
              paymentCount: 2,
              netAfterFees: '+19',
              summary: 'Long Binance, short OKX',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('okx')).toBeInTheDocument();
    expect(screen.getByText('+19')).toBeInTheDocument();
    expect(screen.getByText(/short OKX/)).toBeInTheDocument();
  });

  it('defaults the lookup to the last seven UTC days', () => {
    const range = defaultFundingHistoryRange(new Date('2026-08-08T15:00:00Z'));
    expect(range).toEqual({ from: '2026-08-01', to: '2026-08-08' });
  });
});
