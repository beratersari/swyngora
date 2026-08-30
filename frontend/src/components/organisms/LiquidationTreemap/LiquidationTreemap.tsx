import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  coinBase,
  dominanceFill,
  hoverCardOrigin,
  tileDensity,
  tileInk,
  toTreemapTiles,
} from './LiquidationTreemap.helpers';
import {
  HoverCard,
  Legend,
  LegendBar,
  LegendTick,
  MapFrame,
  Shell,
  TileAmount,
  TileButton,
  TileHost,
  TileSymbol,
  TipHead,
  TipRow,
  TipSym,
} from './LiquidationTreemap.styles';
import type { LiquidationTreemapProps, LiquidationTreemapTile } from './LiquidationTreemap.types';

/**
 * CoinGlass-style liquidation coin map: tile size = notional,
 * color = long vs short mix (red = more longs liquidated).
 */
export function LiquidationTreemap({ coins, isLoading, onOpen }: LiquidationTreemapProps) {
  const { t } = useTranslation('liquidations');
  const { formatCompact } = useDisplayCurrency();
  const frameRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 720, h: 480 });
  const [hover, setHover] = useState<{ tile: LiquidationTreemapTile; x: number; y: number } | null>(
    null,
  );

  useEffect(() => {
    const el = frameRef.current;
    if (!el) return;
    const apply = () => {
      const r = el.getBoundingClientRect();
      if (r.width > 0 && r.height > 0) setSize({ w: r.width, h: r.height });
    };
    apply();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const tiles = useMemo(() => toTreemapTiles(coins, size.w, size.h), [coins, size.h, size.w]);

  const moveTip = useCallback((tile: LiquidationTreemapTile, clientX: number, clientY: number) => {
    const el = frameRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const origin = hoverCardOrigin(clientX - r.left, clientY - r.top, r.width, r.height);
    setHover({ tile, x: origin.x, y: origin.y });
  }, []);

  if (isLoading && coins.length === 0) {
    return <Skeleton height={480} />;
  }
  if (!isLoading && coins.length === 0) {
    return <DeskEmpty title={t('treemap.empty')} hint={t('treemap.emptyHint')} />;
  }

  const tip = hover?.tile;

  return (
    <Shell>
      <MapFrame
        ref={frameRef}
        role="group"
        aria-label={t('treemap.aria')}
        onMouseLeave={() => setHover(null)}
      >
        {tiles.map((tile) => {
          const density = tileDensity(tile.w, tile.h);
          const ticker = coinBase(tile.symbol, tile.base);
          const fill = dominanceFill(tile.longShare);
          const ink = tileInk(fill);
          return (
            <TileHost
              key={tile.symbol}
              $x={(tile.x / size.w) * 100}
              $y={(tile.y / size.h) * 100}
              $w={(tile.w / size.w) * 100}
              $h={(tile.h / size.h) * 100}
            >
              <TileButton
                type="button"
                $fill={fill}
                $ink={ink}
                aria-label={t('treemap.openHeatmap', { symbol: tile.symbol })}
                onClick={() => onOpen?.(tile.symbol)}
                onMouseEnter={(e) => moveTip(tile, e.clientX, e.clientY)}
                onMouseMove={(e) => moveTip(tile, e.clientX, e.clientY)}
              >
                <TileSymbol $size={density}>
                  {density === 'micro' && ticker.length > 5 ? ticker.slice(0, 4) : ticker}
                </TileSymbol>
                {density === 'full' || density === 'compact' ? (
                  <TileAmount $size={density}>{formatCompact(tile.totalNotional, 'USDT')}</TileAmount>
                ) : null}
              </TileButton>
            </TileHost>
          );
        })}
        {tip && hover ? (
          <HoverCard $x={hover.x} $y={hover.y} role="tooltip">
            <TipHead>
              <TipSym>{coinBase(tip.symbol, tip.base)}</TipSym>
            </TipHead>
            <TipRow>
              <span>{t('cards.total')}</span>
              <span>{formatCompact(tip.totalNotional, 'USDT')}</span>
            </TipRow>
            <TipRow>
              <span>{t('cards.long')}</span>
              <span>{formatCompact(tip.longNotional, 'USDT')}</span>
            </TipRow>
            <TipRow>
              <span>{t('cards.short')}</span>
              <span>{formatCompact(tip.shortNotional, 'USDT')}</span>
            </TipRow>
          </HoverCard>
        ) : null}
      </MapFrame>
      <Legend aria-hidden>
        <LegendTick>{t('treemap.legendShort')}</LegendTick>
        <LegendBar />
        <LegendTick>{t('treemap.legendLong')}</LegendTick>
      </Legend>
    </Shell>
  );
}
