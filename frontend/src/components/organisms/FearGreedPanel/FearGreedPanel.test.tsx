import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { FearGreedPanel } from './FearGreedPanel';

describe('FearGreedPanel', () => {
  it('shows the score, the input that moved it, and a past window', () => {
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
    expect(screen.getByText('62')).toBeInTheDocument();
    expect(screen.getByText('Greed')).toBeInTheDocument();
    expect(screen.getByText('Funding moved it most (+8.2 toward greed)')).toBeInTheDocument();
    expect(screen.getByText('OI-weighted funding +0.03%')).toBeInTheDocument();
    expect(screen.getByText('24h 48 → 62 (+14)')).toBeInTheDocument();
    expect(screen.getByText('Past scores')).toBeInTheDocument();
  });

  it('shows an error without a score', () => {
    renderWithProviders(<FearGreedPanel error="Upstream market data unavailable" />);
    expect(screen.getByText('Upstream market data unavailable')).toBeInTheDocument();
    expect(screen.queryByText('Past scores')).not.toBeInTheDocument();
  });
});
