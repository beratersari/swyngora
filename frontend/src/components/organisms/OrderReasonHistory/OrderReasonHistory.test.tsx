import { fireEvent, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { i18n } from '@/libs/i18n';
import { renderWithTheme } from '@/test/render';
import { OrderReasonHistory } from './OrderReasonHistory';
import type { ReasonHistoryRow } from './helpers';

const rows: ReasonHistoryRow[] = [
  {
    clause: 'priceCvd',
    kind: 'price_up_cvd_down',
    still: {
      count: 1,
      profitable: 1,
      realizedPnL: 20,
      closes: [{ exchange: 'binance', symbol: 'BTCUSDT', closedAt: '2026-10-13T12:00:00Z', realizedPnL: 20 }],
    },
    left: {
      count: 1,
      profitable: 0,
      realizedPnL: -10,
      closes: [{ exchange: 'binance', symbol: 'ETHUSDT', closedAt: '2026-10-13T12:01:00Z', realizedPnL: -10 }],
    },
  },
  {
    clause: 'openInterest',
    kind: 'opened',
    still: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
    left: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
  },
];

describe('OrderReasonHistory', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('lists still and left, and names a kind with no past close', () => {
    renderWithTheme(<OrderReasonHistory rows={rows} selected={['priceCvd', 'openInterest']} />);
    expect(screen.getByText('Past closes with this reason')).toBeInTheDocument();
    expect(screen.getByText('Price rising, futures CVD falling')).toBeInTheDocument();
    expect(screen.getByText(/Still the same: 1 close, 1 profitable/)).toBeInTheDocument();
    expect(screen.getByText(/After it left: 1 close, 0 profitable/)).toBeInTheDocument();
    expect(screen.getByText('Open interest building')).toBeInTheDocument();
    expect(screen.getByText('No past close stored this reason.')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Still the same/ }));
    expect(screen.getByText(/BTCUSDT on binance/)).toBeInTheDocument();
  });

  it('hides a clause that is no longer checked', () => {
    renderWithTheme(<OrderReasonHistory rows={rows} selected={['openInterest']} />);
    expect(screen.queryByText(/Still the same: 1 close/)).not.toBeInTheDocument();
    expect(screen.getByText('No past close stored this reason.')).toBeInTheDocument();
  });

  it('stays quiet when nothing is selected', () => {
    renderWithTheme(<OrderReasonHistory rows={rows} selected={[]} />);
    expect(screen.queryByText('Past closes with this reason')).not.toBeInTheDocument();
  });

  it('uses the Turkish lines', async () => {
    await i18n.changeLanguage('tr');
    renderWithTheme(<OrderReasonHistory rows={rows} selected={['openInterest']} />);
    expect(screen.getByText('Bu gerekçedeki geçmiş kapanışlar')).toBeInTheDocument();
    expect(screen.getByText('Bu gerekçeyi saklayan geçmiş bir kapanış yok.')).toBeInTheDocument();
  });
});
