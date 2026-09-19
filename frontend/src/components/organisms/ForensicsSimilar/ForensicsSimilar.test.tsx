import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsSimilar } from './ForensicsSimilar';

describe('ForensicsSimilar', () => {
  it('renders matches and after horizons', () => {
    renderWithProviders(
      <ForensicsSimilar
        similar={{
          summary: '2 similar past setups',
          matches: [
            {
              at: '2026-08-18T12:00:00Z',
              until: '2026-08-18T12:15:00Z',
              direction: 'up',
              similarity: '82',
              coverage: '90',
              afterReturnPct: '+3.5',
              whySimilar: 'Matched most on open interest and volume.',
              compared: [
                { name: 'oi', used: true, matched: true, score: '88' },
                { name: 'volume', used: true, score: '80' },
                { name: 'book', used: false },
              ],
            },
          ],
          afterHorizons: [{ horizon: '1h', up: 2, down: 0, medianPct: '+2.1', sample: 2 }],
        }}
      />,
    );
    expect(screen.getByText(/similar past moves|benzer geçmiş hareketler/i)).toBeInTheDocument();
    expect(screen.getByText('+3.5')).toBeInTheDocument();
    expect(screen.getByText('+2.1')).toBeInTheDocument();
    expect(screen.getByText(/matched most on open interest/i)).toBeInTheDocument();
    expect(screen.getByText(/open interest 88/i)).toBeInTheDocument();
    expect(screen.getByText(/book n\/a/i)).toBeInTheDocument();
  });
});
