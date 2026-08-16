import type { ReactNode } from 'react';
import { formatSymbolDisplay } from '@/libs/utils';
import { tapeCellLabel } from './TickerTape.helpers';
import { Cell, Chg, Px, StaticRow, Strip, Sym, Track } from './TickerTape.styles';
import type { TickerTapeItem, TickerTapeProps } from './TickerTape.types';

export function TickerTapeCell({ item }: { item: TickerTapeItem }) {
  return (
    <Cell to={item.href} aria-label={tapeCellLabel(item)}>
      <Sym>{formatSymbolDisplay(item.symbol)}</Sym>
      <Px>{item.lastPrice}</Px>
      <Chg $up={(item.changeValue ?? 0) > 0} $flat={!item.changeValue}>
        {item.changePercent}
      </Chg>
    </Cell>
  );
}

/** Horizontally scrolling last-price strip. Duplicates items for a seamless loop. */
export function TickerTape({ items, ariaLabel, paused = false }: TickerTapeProps) {
  if (items.length === 0) return null;
  const loop = [...items, ...items];
  return (
    <Track aria-label={ariaLabel} role="region">
      <Strip $paused={paused}>
        {loop.map((item, i) => (
          <TickerTapeCell key={`${item.href}-${i}`} item={item} />
        ))}
      </Strip>
    </Track>
  );
}

/** Non-looping row for a short watchlist — every symbol stays on screen. */
export function TickerTapeStatic({
  ariaLabel,
  children,
}: {
  ariaLabel: string;
  children: ReactNode;
}) {
  return (
    <Track aria-label={ariaLabel} role="region" $scroll>
      <StaticRow>{children}</StaticRow>
    </Track>
  );
}
