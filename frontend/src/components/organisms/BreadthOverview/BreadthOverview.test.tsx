import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { BreadthOverview } from './BreadthOverview';

describe('BreadthOverview', () => {
  it('shows the 1h up share on a split bar and the rest in a dialog', async () => {
    const user = userEvent.setup();
    renderWithTheme(
      <BreadthOverview
        data={{
          universe: 80,
          summary: 'Most coins are up with BTC.',
          windows: [
            {
              window: '1h',
              alignment: 'with_market',
              counts: { upPct: '62', downPct: '28' },
            },
          ],
        }}
      />,
    );
    expect(screen.getByRole('button', { name: /Market breadth/ })).toBeInTheDocument();
    expect(screen.getByText('62%')).toBeInTheDocument();
    expect(screen.getByText('With market')).toBeInTheDocument();
    expect(screen.queryByText('Most coins are up with BTC.')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Market breadth/ }));
    expect(await screen.findByText('62% up · 28% down')).toBeInTheDocument();
    expect(screen.getByText('Most coins are up with BTC.')).toBeInTheDocument();
  });

  it('lists extra windows besides the 1h headline inside the dialog', async () => {
    const user = userEvent.setup();
    renderWithTheme(
      <BreadthOverview
        data={{
          windows: [
            { window: '1h', counts: { upPct: '62', downPct: '28' } },
            { window: '4h', counts: { upPct: '55', downPct: '40' } },
          ],
        }}
      />,
    );
    await user.click(screen.getByRole('button', { name: /Market breadth/ }));
    expect(await screen.findByText('55% up · 40% down')).toBeInTheDocument();
  });
});
