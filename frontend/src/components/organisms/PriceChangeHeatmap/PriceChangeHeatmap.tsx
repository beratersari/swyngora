import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useDisplayCurrency } from '@/libs/hooks';
import { formatChangePercent, marketCapQuote, venueQuote } from '@/libs/utils';
import { HEATMAP_COLOR_CAP_PCT, HEATMAP_LEGEND_GRADIENT } from './PriceChangeHeatmap.constants';
import {
  baseSymbol,
  changeFill,
  formatTileChange,
  hoverCardOrigin,
  tileDensity,
  tileInk,
  toHeatmapTiles,
} from './PriceChangeHeatmap.helpers';
import {
  HoverCard,
  Legend,
  LegendBar,
  LegendTick,
  MapFrame,
  Shell,
  TileButton,
  TileChange,
  TileHost,
  TileSymbol,
  TipChg,
  TipHead,
  TipRow,
  TipSym,
} from './PriceChangeHeatmap.styles';
import type { HeatmapTile, PriceChangeHeatmapProps } from './PriceChangeHeatmap.types';

/**
 * CoinMarketCap-style market treemap: tile size = volume or market cap,
 * color = 24h change. Hover shows a quote card; click opens the pair.
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
  const [hover, setHover] = useState<{ tile: HeatmapTile; x: number; y: number } | null>(null);

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

  const moveTip = useCallback((tile: HeatmapTile, clientX: number, clientY: number) => {
    const el = frameRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const origin = hoverCardOrigin(clientX - r.left, clientY - r.top, r.width, r.height);
    setHover({ tile, x: origin.x, y: origin.y });
  }, []);

  if (isLoading && items.length === 0) {
    return <Skeleton height={520} />;
  }
  if (!isLoading && items.length === 0) {
    return <DeskEmpty title={t('heatmap:empty')} />;
  }

  const tip = hover?.tile;
  const tipFlat = tip ? !Number.isFinite(tip.changePct) || Math.abs(tip.changePct) < 0.005 : true;

  return (
    <Shell>
      <MapFrame
        ref={frameRef}
        role="group"
        aria-label={t('heatmap:mapAria')}
        onMouseLeave={() => setHover(null)}
      >
        {tiles.map((tile) => {
          const density = tileDensity(tile.w, tile.h);
          const ticker = baseSymbol(tile.symbol);
          const change = formatTileChange(tile.changePct);
          const fill = changeFill(tile.changePct);
          const ink = tileInk(fill);
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
                $fill={fill}
                $ink={ink}
                aria-label={`${tile.symbol} ${formatChangePercent(tile.changePct)}`}
                onClick={() => onOpen?.(tile.exchange, tile.symbol)}
                onMouseEnter={(e) => moveTip(tile, e.clientX, e.clientY)}
                onMouseMove={(e) => moveTip(tile, e.clientX, e.clientY)}
                onFocus={() =>
                  setHover({
                    tile,
                    x: tile.x + Math.min(24, tile.w / 2),
                    y: tile.y + Math.min(24, tile.h / 2),
                  })
                }
              >
                <TileSymbol $size={density}>
                  {density === 'micro' && ticker.length > 5 ? ticker.slice(0, 4) : ticker}
                </TileSymbol>
                {density === 'full' || density === 'compact' ? (
                  <TileChange $size={density}>{change}</TileChange>
                ) : null}
              </TileButton>
            </TileHost>
          );
        })}
        {tip && hover ? (
          <HoverCard $x={hover.x} $y={hover.y} role="tooltip">
            <TipHead>
              <TipSym>{baseSymbol(tip.symbol)}</TipSym>
              <TipChg $up={tip.changePct >= 0} $flat={tipFlat}>
                {formatChangePercent(tip.changePct)}
              </TipChg>
            </TipHead>
            <TipRow>
              <span>{t('heatmap:last')}</span>
              <span>{formatPrice(tip.lastPrice, venueQuote(tip.exchange))}</span>
            </TipRow>
            <TipRow>
              <span>{t('heatmap:metric.marketCap')}</span>
              <span>{formatCompact(tip.marketCapCirculating, marketCapQuote(tip.exchange))}</span>
            </TipRow>
            <TipRow>
              <span>{t('heatmap:metric.volume')}</span>
              <span>{formatCompact(tip.quoteVolume, venueQuote(tip.exchange))}</span>
            </TipRow>
            <TipRow>
              <span>{t('heatmap:venue')}</span>
              <span>{tip.exchange}</span>
            </TipRow>
          </HoverCard>
        ) : null}
      </MapFrame>
      <Legend aria-hidden>
        <LegendTick>−{HEATMAP_COLOR_CAP_PCT}%</LegendTick>
        <LegendBar style={{ background: `linear-gradient(90deg, ${HEATMAP_LEGEND_GRADIENT})` }} />
        <LegendTick>+{HEATMAP_COLOR_CAP_PCT}%</LegendTick>
      </Legend>
    </Shell>
  );
}
