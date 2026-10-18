import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { formatDateTime } from '@/libs/utils/formatMarket';
import { renderWithProviders } from '@/test/render';
import { AlertsTable } from './AlertsTable';

const firedAt = '2026-10-13T12:00:00Z';

const situationFire = {
  comparison: {
    same: false,
    clauses: [
      {
        clause: 'priceCvd' as const,
        status: 'changed' as const,
        entry: 'price_up_cvd_down',
        live: 'agrees',
      },
    ],
  },
  entry: { priceCvd: { kind: 'price_up_cvd_down' as const } },
  live: { priceCvd: { kind: 'agrees' as const } },
};

describe('AlertsTable', () => {
  it('renders empty state', () => {
    renderWithProviders(
      <AlertsTable items={[]} loading={false} onDelete={vi.fn()} />,
    );
    expect(screen.getByText(/no alerts|henüz/i)).toBeInTheDocument();
  });

  it('renders a price alert row', () => {
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: '1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            condition: 'above',
            targetPrice: 100,
            mode: 'one_time',
            status: 'active',
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/BTC/)).toBeInTheDocument();
  });

  it('names the venue that crossed an all-exchange funding alert', () => {
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: '2',
            exchange: 'all',
            symbol: 'BTCUSDT',
            kind: 'funding',
            condition: 'above',
            targetPrice: 0.05,
            mode: 'one_time',
            status: 'triggered',
            triggeredPrice: 0.08,
            triggeredVenue: 'bybit',
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/bybit/i)).toBeInTheDocument();
    expect(screen.queryByText(/Last fire/)).not.toBeInTheDocument();
  });

  it('names the last fire, a blank buy, and a flat book', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: 'fire',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            kind: 'situation',
            condition: 'leaves',
            mode: 'repeating',
            status: 'active',
            triggeredAt: firedAt,
            situationWatch: 'blank',
            situationFire,
          },
          {
            id: 'flat',
            exchange: 'binance',
            symbol: 'ETHUSDT',
            kind: 'situation',
            condition: 'leaves',
            mode: 'one_time',
            status: 'active',
            situationWatch: 'flat',
          },
          {
            id: 'price',
            exchange: 'binance',
            symbol: 'SOLUSDT',
            condition: 'above',
            targetPrice: 100,
            mode: 'one_time',
            status: 'active',
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(`Last fire ${formatDateTime(firedAt, 'en')}`)).toBeInTheDocument();
    expect(
      screen.getByText('The newest buy stored no read, so this alert is not watching.'),
    ).toBeInTheDocument();
    expect(screen.getByText('No open buy is left, so this alert stays quiet.')).toBeInTheDocument();
    expect(screen.getByText('≥ 100')).toBeInTheDocument();
    expect(screen.getAllByText('active').length).toBeGreaterThan(1);
    expect(screen.queryByText('The read you entered on is still the live read.')).not.toBeInTheDocument();
  });

  it('keeps the last fire in Turkish and does not treat a matching read as the fire', async () => {
    await i18n.changeLanguage('tr');
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: 'fire-tr',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            kind: 'situation',
            condition: 'leaves',
            mode: 'repeating',
            status: 'active',
            triggeredAt: firedAt,
            situationWatch: 'blank',
            situationFire,
          },
          {
            id: 'same',
            exchange: 'binance',
            symbol: 'ETHUSDT',
            kind: 'situation',
            condition: 'leaves',
            mode: 'repeating',
            status: 'active',
            triggeredAt: firedAt,
            situationFire: {
              comparison: { same: true, clauses: [] },
              entry: { priceCvd: { kind: 'agrees' } },
              live: { priceCvd: { kind: 'agrees' } },
            },
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/Önce:/)).toBeInTheDocument();
    expect(screen.getByText(/Şimdi:/)).toBeInTheDocument();
    expect(screen.getByText(`Son tetiklenme ${formatDateTime(firedAt, 'tr')}`)).toBeInTheDocument();
    expect(
      screen.getByText('En yeni alım bir okuma saklamadı, bu yüzden bu uyarı izlemiyor.'),
    ).toBeInTheDocument();
    expect(screen.queryByText(/hâlâ canlı okuma|still the live read/i)).not.toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('names the selected read and says when it was met', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(
      <AlertsTable
        items={[
          {
            id: 'match',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            kind: 'situation',
            condition: 'matches',
            mode: 'repeating',
            status: 'active',
            triggeredAt: firedAt,
            situationWatch: 'selected',
            situationMatch: { priceCvd: 'price_up_cvd_down', funding: 'long' },
            situationFire: {
              comparison: {
                same: true,
                clauses: [
                  { clause: 'priceCvd', status: 'met', entry: 'price_up_cvd_down', live: 'price_up_cvd_down' },
                  { clause: 'funding', status: 'met', entry: 'long', live: 'long' },
                ],
              },
            },
          },
        ]}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/matches the selected read/i)).toBeInTheDocument();
    expect(screen.getByText(/Repeating/)).toBeInTheDocument();
    expect(
      screen.getByText('Watching Price rising, futures CVD falling; Longs paying funding.'),
    ).toBeInTheDocument();
    expect(screen.getByText(`Last fire ${formatDateTime(firedAt, 'en')}`)).toBeInTheDocument();
    expect(screen.getByText('The selected read was met.')).toBeInTheDocument();
    expect(screen.queryByText(/newest buy stored no read/i)).not.toBeInTheDocument();
  });
});
