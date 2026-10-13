import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { i18n } from '@/libs/i18n';
import { renderWithProviders } from '@/test/render';
import { PortfolioSituationRecord } from './PortfolioSituationRecord';

const closes = [
  {
    exchange: 'binance',
    symbol: 'BTCUSDT',
    closedAt: '2026-10-13T12:05:00.000Z',
    realizedPnL: 20,
  },
  {
    exchange: 'bybit',
    symbol: 'ETHUSDT',
    closedAt: '2026-10-13T12:00:00.000Z',
    realizedPnL: -10,
  },
];

const record = {
  counted: 2,
  skipped: 1,
  rows: [
    {
      clause: 'priceCvd' as const,
      kind: 'price_up_cvd_down',
      count: 2,
      profitable: 1,
      realizedPnL: 10,
      closes,
    },
    {
      clause: 'funding' as const,
      kind: 'long',
      count: 2,
      profitable: 1,
      realizedPnL: 10,
      closes,
    },
  ],
};

describe('PortfolioSituationRecord', () => {
  it('lists each stored kind and does not add the rows into one total', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(<PortfolioSituationRecord record={record} />);
    expect(screen.getByRole('region', { name: 'What your closes made' })).toBeInTheDocument();
    expect(screen.getByText('Price rising, futures CVD falling')).toBeInTheDocument();
    expect(screen.getByText('Longs paying funding')).toBeInTheDocument();
    expect(screen.getAllByText('2 closes, 1 profitable, +10.')).toHaveLength(2);
    expect(
      screen.getByText('1 crypto close had no stored read and is left out.'),
    ).toBeInTheDocument();
    expect(screen.queryByText(/\+20/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Price rising/ })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('opens a row into the closes that stored that kind', async () => {
    await i18n.changeLanguage('en');
    const user = userEvent.setup();
    renderWithProviders(<PortfolioSituationRecord record={record} />);
    await user.click(screen.getByRole('button', { name: /Price rising/ }));
    expect(screen.getByRole('button', { name: /Price rising/ })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(screen.getByText(/BTCUSDT on binance, .+, \+20/)).toBeInTheDocument();
    expect(screen.getByText(/ETHUSDT on bybit, .+, -10/)).toBeInTheDocument();
    expect(screen.getAllByText(/BTCUSDT on binance/)).toHaveLength(1);
    await user.click(screen.getByRole('button', { name: /Longs paying/ }));
    expect(screen.getAllByText(/BTCUSDT on binance, .+, \+20/)).toHaveLength(2);
    expect(screen.getAllByText(/ETHUSDT on bybit, .+, -10/)).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: /Price rising/ }));
    expect(screen.getAllByText(/BTCUSDT on binance/)).toHaveLength(1);
  });

  it('uses the Turkish labels', async () => {
    await i18n.changeLanguage('tr');
    renderWithProviders(<PortfolioSituationRecord record={record} />);
    expect(screen.getByRole('region', { name: 'Kapanışların ne getirdiği' })).toBeInTheDocument();
    expect(screen.getByText('Fiyat yükselirken vadeli CVD düşüyor')).toBeInTheDocument();
    expect(screen.getByText('Fonlamayı longlar ödüyor')).toBeInTheDocument();
    expect(screen.getAllByText('2 kapanış, 1 kârlı, +10.')).toHaveLength(2);
    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /Fiyat yükselirken/ }));
    expect(screen.getByText(/BTCUSDT, binance, .+, \+20/)).toBeInTheDocument();
    expect(screen.getByText(/ETHUSDT, bybit, .+, -10/)).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('says when this book has no stored close yet', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(<PortfolioSituationRecord record={{ counted: 0, skipped: 0, rows: [] }} />);
    expect(
      screen.getByText('No crypto close in this book has a stored read yet.'),
    ).toBeInTheDocument();
  });
});
