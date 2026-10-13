import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { i18n } from '@/libs/i18n';
import { renderWithProviders } from '@/test/render';
import { PortfolioSituationLeave } from './PortfolioSituationLeave';

const stillCloses = [
  {
    exchange: 'binance',
    symbol: 'BTCUSDT',
    closedAt: '2026-10-13T12:00:00.000Z',
    realizedPnL: 20,
  },
];

const leftCloses = [
  {
    exchange: 'bybit',
    symbol: 'ETHUSDT',
    closedAt: '2026-10-13T12:05:00.000Z',
    realizedPnL: -10,
  },
];

const record = {
  counted: 2,
  skipped: 1,
  unreadable: 1,
  rows: [
    {
      clause: 'priceCvd' as const,
      kind: 'price_up_cvd_down',
      still: { count: 1, profitable: 1, realizedPnL: 20, closes: stillCloses },
      left: { count: 1, profitable: 0, realizedPnL: -10, closes: leftCloses },
    },
    {
      clause: 'funding' as const,
      kind: 'long',
      still: { count: 2, profitable: 1, realizedPnL: 10, closes: [...stillCloses, ...leftCloses] },
      left: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
    },
  ],
};

describe('PortfolioSituationLeave', () => {
  it('splits each entry kind into still and left', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(<PortfolioSituationLeave record={record} />);
    expect(screen.getByRole('region', { name: 'Did leaving pay' })).toBeInTheDocument();
    expect(screen.getByText('Price rising, futures CVD falling')).toBeInTheDocument();
    expect(screen.getByText('Longs paying funding')).toBeInTheDocument();
    expect(screen.getByText('Still the same: 1 close, 1 profitable, +20.')).toBeInTheDocument();
    expect(screen.getByText('After it left: 1 close, 0 profitable, -10.')).toBeInTheDocument();
    expect(screen.getByText('Still the same: 2 closes, 1 profitable, +10.')).toBeInTheDocument();
    expect(screen.getByText('After it left: 0 closes, 0 profitable, 0.')).toBeInTheDocument();
    expect(
      screen.getByText('1 crypto close had no stored read on the buy and is left out.'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        '1 crypto close had a stored buy read and no sell read, so it stays out of both groups.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(/\+30/)).not.toBeInTheDocument();
  });

  it('opens one group without opening the other', async () => {
    await i18n.changeLanguage('en');
    const user = userEvent.setup();
    renderWithProviders(<PortfolioSituationLeave record={record} />);
    const still = screen.getByRole('button', { name: /Still the same: 1 close/ });
    expect(still).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('button', { name: /After it left: 0 closes/ })).not.toBeInTheDocument();
    await user.click(still);
    expect(still).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText(/BTCUSDT on binance, .+, \+20/)).toBeInTheDocument();
    expect(screen.queryByText(/ETHUSDT on bybit/)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /After it left: 1 close/ }));
    expect(screen.getByText(/ETHUSDT on bybit, .+, -10/)).toBeInTheDocument();
    expect(screen.getAllByText(/BTCUSDT on binance/)).toHaveLength(1);
  });

  it('uses the Turkish title', async () => {
    await i18n.changeLanguage('tr');
    renderWithProviders(<PortfolioSituationLeave record={record} />);
    expect(screen.getByRole('region', { name: 'Ayrılınca ne oldu' })).toBeInTheDocument();
    expect(screen.getByText('Aynı kaldı: 1 kapanış, 1 kârlı, +20.')).toBeInTheDocument();
    expect(screen.getByText('Ayrıldı: 1 kapanış, 0 kârlı, -10.')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('says when nothing can be compared', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(
      <PortfolioSituationLeave record={{ counted: 0, skipped: 0, unreadable: 0, rows: [] }} />,
    );
    expect(
      screen.getByText('No crypto close in this book can be compared yet.'),
    ).toBeInTheDocument();
  });
});
