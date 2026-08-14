import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { formatChangePercent, formatCompactAmount, formatPrice } from '@/libs/utils';
import {
  baseSymbol,
  changeFill,
  formatTileChange,
  tileDensity,
  toHeatmapTiles,
} from './PriceChangeHeatmap.helpers';
import {
  Hud,
  HudChange,
  HudMeta,
  HudPair,
  MapFrame,
  Shell,
  TileButton,
  TileChange,
  TileHost,
  TilePrice,
  TileSymbol,
} from './PriceChangeHeatmap.styles';
import type { HeatmapTile, PriceChangeHeatmapProps } from './PriceChangeHeatmap.types';

type HudState = { tile: HeatmapTile; x: number; y: number };

/**
 * Full-bleed market map: opaque red / slate / green on a charcoal well.
 */
export function PriceChangeHeatmap({
  items,
  metric = 'quoteVolume',
  isLoading,
  onOpen,
}: PriceChangeHeatmapProps) {
  const { t } = useTranslation(['heatmap', 'common']);
  const frameRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 960, h: 560 });
  const [hud, setHud] = useState<HudState | null>(null);

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

  const tiles = useMemo(
    () => toHeatmapTiles(items, metric, size.w, size.h),
    [items, metric, size.h, size.w],
  );

  const placeHud = useCallback((tile: HeatmapTile, clientX: number, clientY: number) => {
    const el = frameRef.current;
    if (!el) return;
    const box = el.getBoundingClientRect();
    let x = clientX - box.left + 14;
    let y = clientY - box.top + 16;
    if (x > box.width - 180) x = clientX - box.left - 168;
    if (y > box.height - 96) y = clientY - box.top - 88;
    setHud({ tile, x: Math.max(8, x), y: Math.max(8, y) });
  }, []);

  if (isLoading && items.length === 0) {
    return <Skeleton height={520} />;
  }
  if (!isLoading && items.length === 0) {
    return <DeskEmpty title={t('heatmap:empty')} />;
  }

  return (
    <Shell>
      <MapFrame
        ref={frameRef}
        role="group"
        aria-label={t('heatmap:mapAria')}
        onMouseLeave={() => setHud(null)}
      >
        {tiles.map((tile) => {
          const density = tileDensity(tile.w, tile.h);
          const ticker = baseSymbol(tile.symbol);
          const change = formatTileChange(tile.changePct);
          return (
            <TileHost
              key={`${tile.exchange}-${tile.symbol}`}
              $x={(tile.x / size.w) * 100}
              $y={(tile.y / size.h) * 100}
              $w={(tile.w / size.w) * 100}
              $h={(tile.h / size.h) * 100}
            >
              <TileButton
                type="button"
                $fill={changeFill(tile.changePct)}
                aria-label={`${tile.symbol} ${formatChangePercent(tile.changePct)}`}
                onClick={() => onOpen?.(tile.exchange, tile.symbol)}
                onMouseMove={(e) => placeHud(tile, e.clientX, e.clientY)}
                onFocus={(e) => {
                  const r = e.currentTarget.getBoundingClientRect();
                  placeHud(tile, r.left + r.width / 2, r.top + r.height / 2);
                }}
              >
                <TileSymbol $size={density}>
                  {density === 'micro' && ticker.length > 5 ? ticker.slice(0, 4) : ticker}
                </TileSymbol>
                {density === 'full' || density === 'compact' ? (
                  <TileChange>{change}</TileChange>
                ) : null}
                {density === 'full' ? <TilePrice>{formatPrice(tile.lastPrice)}</TilePrice> : null}
              </TileButton>
            </TileHost>
          );
        })}
        {hud ? (
          <Hud $x={hud.x} $y={hud.y}>
            <HudPair>{hud.tile.symbol}</HudPair>
            <HudChange>{formatChangePercent(hud.tile.changePct)}</HudChange>
            <HudMeta>{formatPrice(hud.tile.lastPrice)}</HudMeta>
            <HudMeta>
              {metric === 'marketCap'
                ? formatCompactAmount(hud.tile.marketCapCirculating)
                : formatCompactAmount(hud.tile.quoteVolume)}
            </HudMeta>
          </Hud>
        ) : null}
      </MapFrame>
    </Shell>
  );
}
