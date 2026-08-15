import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useDisplayCurrency } from '@/libs/hooks';
import { formatChangePercent, marketCapQuote, venueQuote } from '@/libs/utils';
import { HEATMAP_LEGEND_GRADIENT } from './PriceChangeHeatmap.constants';
import {
  baseSymbol,
  changeFill,
  formatTileChange,
  tileDensity,
  toHeatmapTiles,
} from './PriceChangeHeatmap.helpers';
import {
  Board,
  Inspector,
  InspectorChange,
  InspectorDd,
  InspectorDt,
  InspectorHint,
  InspectorKicker,
  InspectorMeta,
  InspectorPair,
  InspectorRow,
  MapFrame,
  Scale,
  ScaleBar,
  ScaleTick,
  Shell,
  TileButton,
  TileChange,
  TileHost,
  TilePrice,
  TileSymbol,
} from './PriceChangeHeatmap.styles';
import type { HeatmapTile, PriceChangeHeatmapProps } from './PriceChangeHeatmap.types';

/**
 * Full-viewport market map with a docked quote inspector.
 */
export function PriceChangeHeatmap({
  items,
  metric = 'quoteVolume',
  isLoading,
  onOpen,
}: PriceChangeHeatmapProps) {
  const { t } = useTranslation(['heatmap', 'common']);
  const { formatPrice, formatCompact } = useDisplayCurrency();
  const frameRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 960, h: 560 });
  const [active, setActive] = useState<HeatmapTile | null>(null);

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

  useEffect(() => {
    if (!tiles.length) {
      setActive(null);
      return;
    }
    setActive((cur) => {
      if (cur) {
        const next = tiles.find((tile) => tile.symbol === cur.symbol && tile.exchange === cur.exchange);
        if (next) return next;
      }
      return tiles[0] ?? null;
    });
  }, [tiles]);

  const pick = useCallback((tile: HeatmapTile) => {
    setActive(tile);
  }, []);

  if (isLoading && items.length === 0) {
    return <Skeleton height={520} />;
  }
  if (!isLoading && items.length === 0) {
    return <DeskEmpty title={t('heatmap:empty')} />;
  }

  const chg = active?.changePct ?? 0;
  const flat = !Number.isFinite(chg) || Math.abs(chg) < 0.005;

  return (
    <Shell>
      <Board>
        <Scale aria-hidden>
          <ScaleTick>+8%</ScaleTick>
          <ScaleBar style={{ background: `linear-gradient(180deg, ${HEATMAP_LEGEND_GRADIENT})` }} />
          <ScaleTick>−8%</ScaleTick>
        </Scale>
        <MapFrame ref={frameRef} role="group" aria-label={t('heatmap:mapAria')}>
          {tiles.map((tile) => {
            const density = tileDensity(tile.w, tile.h);
            const ticker = baseSymbol(tile.symbol);
            const change = formatTileChange(tile.changePct);
            const on = active?.symbol === tile.symbol && active.exchange === tile.exchange;
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
                  $on={on}
                  aria-label={`${tile.symbol} ${formatChangePercent(tile.changePct)}`}
                  aria-pressed={on}
                  onClick={() => {
                    pick(tile);
                    onOpen?.(tile.exchange, tile.symbol);
                  }}
                  onMouseEnter={() => pick(tile)}
                  onFocus={() => pick(tile)}
                >
                  <TileSymbol $size={density}>
                    {density === 'micro' && ticker.length > 5 ? ticker.slice(0, 4) : ticker}
                  </TileSymbol>
                  {density === 'full' || density === 'compact' ? <TileChange>{change}</TileChange> : null}
                  {density === 'full' ? (
                    <TilePrice>{formatPrice(tile.lastPrice, venueQuote(tile.exchange))}</TilePrice>
                  ) : null}
                </TileButton>
              </TileHost>
            );
          })}
        </MapFrame>
      </Board>
      <Inspector>
        <InspectorKicker>{t('heatmap:inspector', { defaultValue: 'Selected' })}</InspectorKicker>
        {active ? (
          <>
            <InspectorPair>{active.symbol}</InspectorPair>
            <InspectorChange $up={chg >= 0} $flat={flat}>
              {formatChangePercent(active.changePct)}
            </InspectorChange>
            <InspectorMeta>
              <InspectorRow>
                <InspectorDt>{t('heatmap:last', { defaultValue: 'Last' })}</InspectorDt>
                <InspectorDd>{formatPrice(active.lastPrice, venueQuote(active.exchange))}</InspectorDd>
              </InspectorRow>
              <InspectorRow>
                <InspectorDt>
                  {metric === 'marketCap'
                    ? t('heatmap:metric.marketCap')
                    : t('heatmap:metric.volume')}
                </InspectorDt>
                <InspectorDd>
                  {metric === 'marketCap'
                    ? formatCompact(active.marketCapCirculating, marketCapQuote(active.exchange))
                    : formatCompact(active.quoteVolume, venueQuote(active.exchange))}
                </InspectorDd>
              </InspectorRow>
              <InspectorRow>
                <InspectorDt>{t('heatmap:venue', { defaultValue: 'Venue' })}</InspectorDt>
                <InspectorDd>{active.exchange}</InspectorDd>
              </InspectorRow>
            </InspectorMeta>
          </>
        ) : (
          <InspectorHint>{t('heatmap:hoverHint', { defaultValue: 'Hover a tile.' })}</InspectorHint>
        )}
      </Inspector>
    </Shell>
  );
}
