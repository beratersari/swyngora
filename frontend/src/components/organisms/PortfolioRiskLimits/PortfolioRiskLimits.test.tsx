import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PortfolioRiskLimits } from './PortfolioRiskLimits';

const save = vi.fn(() => ({ unwrap: () => Promise.resolve({}) }));

vi.mock('@/libs/api', () => ({
  useGetPortfolioRiskLimitsQuery: () => ({
    data: {
      limits: { maxDailyLossPct: 5, maxAssetWeightPct: 30, maxExposurePct: 60 },
      status: { exposurePct: 72.5, exposureLimitHit: true },
    },
  }),
  useSetPortfolioRiskLimitsMutation: () => [save, { isLoading: false }],
}));

describe('PortfolioRiskLimits', () => {
  it('shows total exposure against the cap and keeps the other rules on save', async () => {
    const user = userEvent.setup();
    renderWithTheme(<PortfolioRiskLimits portfolioId="b1" />);
    expect(screen.getByText('Now 72.5% · 60%')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(save).toHaveBeenCalledWith({
      portfolioId: 'b1',
      maxDailyLossPct: 5,
      maxAssetWeightPct: 30,
      maxExposurePct: 60,
    });
  });
});
