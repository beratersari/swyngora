import type { OrderCountCoin, OrderCountMarketWindow } from '@/libs/api';
import type { OrderCountRow } from '@/components/organisms/OrderCountTable';
import type { OrderCountWindowId } from './constants';

export function windowById(
  windows: OrderCountMarketWindow[] | undefined,
  id: OrderCountWindowId,
): OrderCountMarketWindow | undefined {
  return windows?.find((w) => w.window === id);
}

export function coinsToRows(
  coins: OrderCountCoin[] | undefined,
  windowId: OrderCountWindowId,
  leanLabel: (lean: string) => string,
): OrderCountRow[] {
  const rows = (coins ?? []).map((coin) => {
    const win = coin.windows?.find((w) => w.window === windowId);
    return {
      exchange: coin.exchange || 'binance',
      symbol: coin.symbol || '',
      ratio: win?.ratio || '—',
      lean: leanLabel(win?.lean || ''),
      buyCount: win?.buyCount === undefined ? '—' : String(win.buyCount),
      sellCount: win?.sellCount === undefined ? '—' : String(win.sellCount),
      summary: win?.summary || coin.summary || '',
      sortRatio: Number.parseFloat(win?.ratio || '0') || 0,
    };
  });
  rows.sort((a, b) => b.sortRatio - a.sortRatio || a.symbol.localeCompare(b.symbol));
  return rows.map(({ sortRatio: _sort, ...row }) => row);
}
