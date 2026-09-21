import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { FearGreedPanel } from './FearGreedPanel';

describe('FearGreedPanel', () => {
  it('keeps the page to the score and opens drivers and history in a dialog', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <FearGreedPanel
        data={{
          score: 62,
          level: 'greed',
          label: 'Greed',
          used: 7,
          total: 8,
          top: { id: 'funding', name: 'Funding', pull: '+8.2', toward: 'greed', detail: 'OI-weighted funding +0.03%' },
          factors: [
            { id: 'funding', name: 'Funding', pull: '+8.2', toward: 'greed', detail: 'OI-weighted funding +0.03%' },
          ],
          windows: [{ window: '24h', pastScore: 48, score: 62, change: 14, complete: true }],
          history: [
            { time: '2026-09-21T11:00:00Z', score: 48, level: 'fear' },
            { time: '2026-09-21T12:00:00Z', score: 62, level: 'greed' },
          ],
          summary: 'Greed 62. Funding moved the score the most (+8.2 toward greed).',
        }}
      />,
    );
    expect(screen.getByRole('button', { name: /Fear & Greed/ })).toBeInTheDocument();
    expect(screen.getByText('62')).toBeInTheDocument();
    expect(screen.getByText('Greed')).toBeInTheDocument();
    expect(screen.queryByText('OI-weighted funding +0.03%')).not.toBeInTheDocument();
    expect(screen.queryByText(/Not the Binance or CoinGlass index/)).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Fear & Greed/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('OI-weighted funding +0.03%')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'What moved it' })).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Past scores' }));
    expect(screen.getByText('24h 48 → 62 (+14)')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'How this score is calculated' }));
    expect(screen.getByText(/Not the Binance or CoinGlass index/)).toBeInTheDocument();
  });

  it('keeps an error off the page until the chip is opened', async () => {
    const user = userEvent.setup();
    renderWithProviders(<FearGreedPanel error="Upstream market data unavailable" />);
    expect(screen.queryByText('Upstream market data unavailable')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Fear & Greed/ }));
    expect(await screen.findByText('Upstream market data unavailable')).toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Past scores' })).not.toBeInTheDocument();
  });
});
