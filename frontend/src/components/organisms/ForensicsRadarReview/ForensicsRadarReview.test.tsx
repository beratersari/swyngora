import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { ForensicsRadarReview } from './ForensicsRadarReview';

describe('ForensicsRadarReview', () => {
  it('renders usefulness and after cells', async () => {
    const onOpen = vi.fn();
    renderWithProviders(
      <ForensicsRadarReview
        review={{
          summary: 'Saved 1 setup(s) over 7d. after 1h: 1 of 1 directional calls matched (100%).',
          cases: [
            {
              id: 'c1',
              symbol: 'BTCUSDT',
              foundAt: '2026-09-04T12:00:00Z',
              similarity: '82',
              lean: 'up',
              whySimilar: 'Closest past twin matched most on open interest.',
              after: [
                { horizon: '1h', filledAt: '2026-09-04T13:00:00Z', returnPct: '+1.2', direction: 'up', useful: true },
              ],
            },
          ],
          horizons: [{ horizon: '1h', sample: 1, leanRight: 1, leanWrong: 0, usefulPct: '100', medianPct: '+1.2' }],
        }}
        onOpenCase={onOpen}
      />,
    );
    expect(screen.getByText(/Saved 1 setup/i)).toBeInTheDocument();
    expect(screen.getByText('BTCUSDT')).toBeInTheDocument();
    expect(screen.getByText(/\+1\.2 ↑ ✓/)).toBeInTheDocument();
    await userEvent.click(screen.getByText('BTCUSDT'));
    expect(onOpen).toHaveBeenCalledWith('BTCUSDT', '2026-09-04T12:00:00Z');
  });
});
