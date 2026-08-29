import { Spin } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { LIQ_HEATMAP_RANGES } from './LiquidationHeatmap.constants';
import {
  buildLayout,
  drawHeatmap,
  formatHitRate,
  formatLiqNotional,
  formatLiqPrice,
  formatLiqTime,
  formatLookahead,
  hasHeatTape,
  hitTest,
  pickGrid,
  pickReview,
} from './LiquidationHeatmap.helpers';
import {
  Chip,
  ChipRow,
  HeatCanvas,
  HoverCard,
  MapFrame,
  Panel,
  ReviewTable,
  ScaleBar,
  ScaleLegend,
  TitleRow,
} from './LiquidationHeatmap.styles';
import type {
  LiqHeatHover,
  LiqHeatRange,
  LiqHeatSide,
  LiqHeatVenue,
  LiquidationHeatmapProps,
} from './LiquidationHeatmap.types';

const VENUES: LiqHeatVenue[] = ['combined', 'binance', 'bybit'];
const SIDES: LiqHeatSide[] = ['totals', 'longs', 'shorts'];

/**
 * CoinGlass-style price × time liquidation intensity. Time left→right,
 * highest price at the top. Color is estimated notional in that cell.
 */
export function LiquidationHeatmap({
  data,
  range,
  onRangeChange,
  venue,
  onVenueChange,
  side,
  onSideChange,
  lastPrice,
  isLoading,
  isFetching,
  errorMessage,
}: LiquidationHeatmapProps) {
  const { t } = useTranslation('detail');
  const frameRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [size, setSize] = useState({ w: 720, h: 400 });
  const [hover, setHover] = useState<LiqHeatHover | null>(null);
  const hasTape = hasHeatTape(data);
  const grid = pickGrid(data, venue);
  const review = pickReview(data, venue);
  const reviewRows = review?.horizons ?? [];

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

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
    canvas.width = Math.max(1, Math.floor(size.w * dpr));
    canvas.height = Math.max(1, Math.floor(size.h * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.imageSmoothingEnabled = false;
    const layout = buildLayout(data, size.w, size.h);
    if (!layout) {
      ctx.fillStyle = '#0B1220';
      ctx.fillRect(0, 0, size.w, size.h);
      return;
    }
    drawHeatmap(ctx, data, layout, venue, side, lastPrice, range);
  }, [data, lastPrice, range, side, size.h, size.w, venue]);

  const onMove = useCallback(
    (event: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const r = canvas.getBoundingClientRect();
      const layout = buildLayout(data, r.width, r.height);
      if (!layout) {
        setHover(null);
        return;
      }
      setHover(hitTest(event.clientX - r.left, event.clientY - r.top, data, layout, venue));
    },
    [data, venue],
  );

  const coveragePct =
    grid && Number.isFinite(grid.coverage) ? Math.round((grid.coverage ?? 0) * 100) : null;

  return (
    <Panel data-testid="liquidation-heatmap">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('liqHeatmap.title')}
        </Text>
        {isFetching && !isLoading ? <Spin size="small" /> : null}
      </TitleRow>
      <Text variant="caption" color="secondary">
        {t('liqHeatmap.subtitle')}
      </Text>
      <ChipRow>
        <Text variant="caption" color="secondary">
          {t('liqHeatmap.range')}
        </Text>
        {LIQ_HEATMAP_RANGES.map((r) => (
          <Chip
            key={r}
            type="button"
            $active={r === range}
            onClick={() => onRangeChange(r as LiqHeatRange)}
          >
            {t(`liqHeatmap.ranges.${r}`)}
          </Chip>
        ))}
        <Text variant="caption" color="secondary">
          {t('liqHeatmap.venue')}
        </Text>
        {VENUES.map((v) => (
          <Chip key={v} type="button" $active={v === venue} onClick={() => onVenueChange(v)}>
            {t(`liqHeatmap.venues.${v}`)}
          </Chip>
        ))}
        <Text variant="caption" color="secondary">
          {t('liqHeatmap.side')}
        </Text>
        {SIDES.map((s) => (
          <Chip key={s} type="button" $active={s === side} onClick={() => onSideChange(s)}>
            {t(`liqHeatmap.sides.${s}`)}
          </Chip>
        ))}
        {coveragePct !== null ? (
          <Text variant="caption" color="secondary" data-testid="liquidation-heatmap-coverage">
            {t('liqHeatmap.coverage', { pct: coveragePct })}
          </Text>
        ) : null}
        <ScaleLegend>
          <Text variant="caption" color="secondary">
            {t('liqHeatmap.low')}
          </Text>
          <ScaleBar $tone={side} />
          <Text variant="caption" color="secondary">
            {t('liqHeatmap.high')}
          </Text>
        </ScaleLegend>
      </ChipRow>

      {errorMessage ? (
        <Text variant="body" color="secondary">
          {errorMessage}
        </Text>
      ) : isLoading && !hasTape ? (
        <Skeleton height={360} />
      ) : !hasTape ? (
        <Text variant="body" color="secondary">
          {t('liqHeatmap.empty')}
        </Text>
      ) : (
        <MapFrame ref={frameRef}>
          <HeatCanvas
            ref={canvasRef}
            data-testid="liquidation-heatmap-canvas"
            onMouseMove={onMove}
            onMouseLeave={() => setHover(null)}
          />
          {hover ? (
            <HoverCard
              $x={Math.min(hover.x + 12, size.w - 196)}
              $y={Math.max(8, hover.y - 92)}
              data-testid="liquidation-heatmap-hover"
            >
              <Text variant="caption" color="secondary">
                {formatLiqTime(hover.timeMs, range)} ·{' '}
                {formatLiqPrice(hover.price, data?.priceStep ?? 0)}
              </Text>
              <Text variant="caption" color="primary">
                {t('liqHeatmap.total')}: {formatLiqNotional(hover.totals)}
              </Text>
              <Text variant="caption" color="secondary">
                {t('liqHeatmap.longs')}: {formatLiqNotional(hover.longs)}
              </Text>
              <Text variant="caption" color="secondary">
                {t('liqHeatmap.shorts')}: {formatLiqNotional(hover.shorts)}
              </Text>
            </HoverCard>
          ) : null}
        </MapFrame>
      )}

      {reviewRows.length > 0 ? (
        <>
          <Text variant="h4" color="primary">
            {t('liqHeatmap.review.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('liqHeatmap.review.subtitle')}
          </Text>
          <ReviewTable data-testid="liquidation-heatmap-review">
            <thead>
              <tr>
                <th>{t('liqHeatmap.review.horizon')}</th>
                <th>{t('liqHeatmap.review.signals')}</th>
                <th>{t('liqHeatmap.review.validated')}</th>
                <th>{t('liqHeatmap.review.missing')}</th>
                <th>{t('liqHeatmap.review.priceMissing')}</th>
                <th>{t('liqHeatmap.review.liqMissing')}</th>
                <th>{t('liqHeatmap.review.hits')}</th>
                <th>{t('liqHeatmap.review.falseSignals')}</th>
                <th>{t('liqHeatmap.review.hitRate')}</th>
                <th>{t('liqHeatmap.review.avgTime')}</th>
                <th>{t('liqHeatmap.review.liqRose')}</th>
              </tr>
            </thead>
            <tbody>
              {reviewRows.map((row) => (
                <tr key={row.horizon ?? 'h'}>
                  <td>{row.horizon ?? '—'}</td>
                  <td>{row.signals ?? 0}</td>
                  <td>{row.validated ?? 0}</td>
                  <td>{row.missing ?? 0}</td>
                  <td>{row.priceMissing ?? 0}</td>
                  <td>{row.liqMissing ?? 0}</td>
                  <td>{row.hits ?? 0}</td>
                  <td>{row.falseSignals ?? 0}</td>
                  <td>{row.validated ? formatHitRate(row.hitRate) : '—'}</td>
                  <td>{formatLookahead(row.avgTimeToHitSec)}</td>
                  <td>
                    {row.hits
                      ? `${row.liqIncreased ?? 0}/${row.hits} (${formatHitRate(row.liqIncreaseRate)})`
                      : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </ReviewTable>
        </>
      ) : null}
    </Panel>
  );
}
