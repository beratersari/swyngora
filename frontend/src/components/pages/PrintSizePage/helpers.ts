import type { PrintSizeCoin, PrintSizeMarketWindow } from '@/libs/api';
import type { PrintSizeRow } from '@/components/organisms/PrintSizeTable';
import type { PrintSizeWindowId } from './constants';

export function windowById(
  windows: PrintSizeMarketWindow[] | undefined,
  id: PrintSizeWindowId,
): PrintSizeMarketWindow | undefined {
  return windows?.find((w) => w.window === id);
}

export function coinsToRows(
  coins: PrintSizeCoin[] | undefined,
  windowId: PrintSizeWindowId,
  readLabel: (read: string) => string,
): PrintSizeRow[] {
  const rows = (coins ?? []).map((coin) => {
    const win = coin.windows?.find((w) => w.window === windowId);
    const buyRatio = Number.parseFloat(win?.buy?.ratio || '0') || 0;
    const sellRatio = Number.parseFloat(win?.sell?.ratio || '0') || 0;
    return {
      exchange: coin.exchange || 'binance',
      symbol: coin.symbol || '',
      buyAvg: win?.buy?.avg || '—',
      buyPrior: win?.buy?.priorAvg || '—',
      sellAvg: win?.sell?.avg || '—',
      sellPrior: win?.sell?.priorAvg || '—',
      read: readLabel(win?.read || ''),
      sortRatio: Math.max(buyRatio, sellRatio),
    };
  });
  rows.sort((a, b) => b.sortRatio - a.sortRatio || a.symbol.localeCompare(b.symbol));
  return rows.map(({ sortRatio: _sort, ...row }) => row);
}
