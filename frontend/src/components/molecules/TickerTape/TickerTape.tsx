import { formatSymbolDisplay } from '@/libs/utils';
import { tapeCellLabel } from './TickerTape.helpers';
import { Cell, Chg, Px, Strip, Sym, Track } from './TickerTape.styles';
import type { TickerTapeProps } from './TickerTape.types';

/** Horizontally scrolling last-price strip. Duplicates items for a seamless loop. */
export function TickerTape({ items, ariaLabel, paused = false }: TickerTapeProps) {
  if (items.length === 0) return null;
  const loop = [...items, ...items];
  return (
    <Track aria-label={ariaLabel} role="region">
      <Strip $paused={paused}>
        {loop.map((item, i) => (
          <Cell key={`${item.href}-${i}`} to={item.href} aria-label={tapeCellLabel(item)}>
            <Sym>{formatSymbolDisplay(item.symbol)}</Sym>
            <Px>{item.lastPrice}</Px>
            <Chg $up={(item.changeValue ?? 0) > 0} $flat={!item.changeValue}>
              {item.changePercent}
            </Chg>
          </Cell>
        ))}
      </Strip>
    </Track>
  );
}
