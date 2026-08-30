import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { LiquidationCascade } from './LiquidationCascade';

describe('LiquidationCascade', () => {
  it('renders venue cards and both-venues note', () => {
    renderWithProviders(
      <LiquidationCascade
        report={{
          symbol: 'BTCUSDT',
          summary: 'Same long cascade on Binance and Bybit.',
          both: {
            agree: true,
            side: 'long',
            grade: 'cascade',
            summary: 'Same long cascade on Binance and Bybit (cascade, score 80).',
          },
          venues: [
            {
              exchange: 'binance',
              symbol: 'BTCUSDT',
              grade: 'cascade',
              side: 'long',
              score: 80,
              hottest: '1m',
              summary: 'binance BTCUSDT cascade long burst',
              windows: [{ window: '1m', longNotional: '12000', shortNotional: '100', maxRatio: 8, grade: 'cascade' }],
            },
            {
              exchange: 'bybit',
              symbol: 'BTCUSDT',
              grade: 'cascade',
              side: 'long',
              score: 75,
              hottest: '1m',
              summary: 'bybit BTCUSDT cascade long burst',
              windows: [{ window: '1m', longNotional: '9000', shortNotional: '80', maxRatio: 7, grade: 'cascade' }],
            },
          ],
          episodes: [
            {
              exchange: 'both',
              combined: true,
              side: 'long',
              grade: 'cascade',
              startedAt: '2026-08-30T15:21:00.000Z',
              durationSec: 600,
              longNotional: '20000',
              shortNotional: '100',
              priceChangePct: '-1.20',
              open: false,
            },
          ],
        }}
        hits={[{ symbol: 'SOLUSDT', side: 'long', grade: 'elevated', score: 40, hottest: '5m', both: false }]}
      />,
    );
    expect(screen.getByTestId('liquidation-cascade')).toBeInTheDocument();
    expect(screen.getAllByText(/same long cascade/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/binance/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/bybit/i).length).toBeGreaterThan(0);
    expect(screen.getByText('SOLUSDT')).toBeInTheDocument();
    expect(screen.getByText(/15:21/)).toBeInTheDocument();
    expect(screen.getByText(/-1\.20%/)).toBeInTheDocument();
  });

  it('opens a bursting coin from the hits list', async () => {
    const onOpenCoin = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LiquidationCascade
        report={{ symbol: 'all', summary: 'Market is quiet.', venues: [] }}
        hits={[{ symbol: 'ETHUSDT', side: 'short', grade: 'cascade', score: 60, hottest: '1m', both: true }]}
        onOpenCoin={onOpenCoin}
      />,
    );
    await user.click(screen.getByRole('button', { name: /ETH/i }));
    expect(onOpenCoin).toHaveBeenCalledWith('ETHUSDT');
  });
});
