import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsTape } from './ForensicsTape';

describe('ForensicsTape', () => {
  it('renders three phases and a change row', () => {
    renderWithProviders(
      <ForensicsTape
        venue={{
          exchange: 'combined',
          phases: [
            { phase: 'before', complete: true, price: { close: '100' } },
            { phase: 'during', complete: true, price: { close: '107' } },
            { phase: 'after', complete: true, price: { close: '105' } },
          ],
          changes: [{ metric: 'price', before: '100', during: '107', after: '105', path: 'faded' }],
        }}
      />,
    );
    expect(screen.getAllByText(/before|önce/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/during|sırada/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText('107').length).toBeGreaterThan(0);
    expect(screen.getByText('faded')).toBeInTheDocument();
  });

  it('shows unusual options on a phase', () => {
    renderWithProviders(
      <ForensicsTape
        venue={{
          exchange: 'combined',
          phases: [
            {
              phase: 'before',
              complete: true,
              options: {
                complete: true,
                unusual: true,
                started: 'before',
                lead: '40m',
                together: 2,
                oiChangePct: '+8.00%',
                oiDirection: 'up',
                leanTo: 'puts',
              },
            },
            { phase: 'during', complete: true },
            { phase: 'after', complete: true },
          ],
        }}
      />,
    );
    expect(screen.getByText(/unusual/i)).toBeInTheDocument();
    expect(screen.getByText(/\+8\.00%/)).toBeInTheDocument();
  });

  it('shows which venue started the price move first', () => {
    renderWithProviders(
      <ForensicsTape
        venue={{ exchange: 'combined', phases: [{ phase: 'during', complete: true }] }}
        venueTiming={{
          agreement: 'binance_earlier',
          follow: 'followed',
          summary: 'Binance started the price move about 8 minutes before Bybit. Bybit followed later.',
        }}
      />,
    );
    expect(screen.getByText(/binance started the price move/i)).toBeInTheDocument();
    expect(screen.getByText(/bybit followed later/i)).toBeInTheDocument();
  });

  it('says when the move stayed on Binance', () => {
    renderWithProviders(
      <ForensicsTape
        venue={{ exchange: 'combined', phases: [{ phase: 'during', complete: true }] }}
        venueTiming={{
          agreement: 'binance_only',
          follow: 'stayed',
          summary: 'The price move printed on Binance. Bybit did not follow.',
        }}
      />,
    );
    expect(screen.getByText(/bybit did not follow/i)).toBeInTheDocument();
  });
});
