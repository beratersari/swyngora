import { Spin } from 'antd';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import {
  DEPTH_ASK_FILL,
  DEPTH_ASK_STROKE,
  DEPTH_AXIS,
  DEPTH_BID_FILL,
  DEPTH_BID_STROKE,
  DEPTH_CHART_HEIGHT,
  DEPTH_GRID,
  DEPTH_MID,
  DEPTH_PLOT_BG,
} from './OrderDepthChart.constants';
import {
  buildDepthLayout,
  buildDepthSeries,
  depthToY,
  formatDepthAmount,
  formatDepthPrice,
  hitTestDepth,
  priceToX,
  xTicks,
  yTicks,
} from './OrderDepthChart.helpers';
import {
  ChartFrame,
  DepthCanvas,
  HoverCard,
  MetricChip,
  MetricRow,
  Panel,
  TitleRow,
} from './OrderDepthChart.styles';
import type { DepthHover, DepthMetric, OrderDepthChartProps } from './OrderDepthChart.types';

/**
 * Classic market-depth graph: cumulative bids (green, left of mid) and
 * asks (red, right of mid) from the live grouped book.
 */
export function OrderDepthChart({
  book,
  isLoading,
  isFetching,
  errorMessage,
}: OrderDepthChartProps) {
  const { t } = useTranslation('detail');
  const frameRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [size, setSize] = useState({ w: 720, h: DEPTH_CHART_HEIGHT });
  const [metric, setMetric] = useState<DepthMetric>('base');
  const [hover, setHover] = useState<DepthHover | null>(null);

  const series = useMemo(() => buildDepthSeries(book, metric), [book, metric]);

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
    if (!canvas || !series) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
    canvas.width = Math.max(1, Math.floor(size.w * dpr));
    canvas.height = Math.max(1, Math.floor(size.h * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.imageSmoothingEnabled = true;
    ctx.fillStyle = DEPTH_PLOT_BG;
    ctx.fillRect(0, 0, size.w, size.h);

    const layout = buildDepthLayout(series, size.w, size.h);
    const floorY = layout.plotY + layout.plotH;
    const midX = priceToX(series.mid, layout);

    ctx.strokeStyle = DEPTH_GRID;
    ctx.lineWidth = 1;
    for (const tick of yTicks(series.maxDepth)) {
      const y = depthToY(tick, layout);
      ctx.beginPath();
      ctx.moveTo(layout.plotX, y);
      ctx.lineTo(layout.plotX + layout.plotW, y);
      ctx.stroke();
    }

    const fillSide = (points: typeof series.bids, fill: string, stroke: string) => {
      if (!points.length) return;
      ctx.beginPath();
      ctx.moveTo(priceToX(points[0].price, layout), floorY);
      for (const p of points) {
        ctx.lineTo(priceToX(p.price, layout), depthToY(p.depth, layout));
      }
      ctx.lineTo(priceToX(points[points.length - 1].price, layout), floorY);
      ctx.closePath();
      ctx.fillStyle = fill;
      ctx.fill();
      ctx.beginPath();
      ctx.moveTo(priceToX(points[0].price, layout), depthToY(points[0].depth, layout));
      for (const p of points) {
        ctx.lineTo(priceToX(p.price, layout), depthToY(p.depth, layout));
      }
      ctx.strokeStyle = stroke;
      ctx.lineWidth = 1.5;
      ctx.stroke();
    };

    fillSide(series.bids, DEPTH_BID_FILL, DEPTH_BID_STROKE);
    fillSide(series.asks, DEPTH_ASK_FILL, DEPTH_ASK_STROKE);

    ctx.strokeStyle = DEPTH_MID;
    ctx.lineWidth = 1;
    ctx.setLineDash([4, 3]);
    ctx.beginPath();
    ctx.moveTo(midX, layout.plotY);
    ctx.lineTo(midX, floorY);
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.fillStyle = DEPTH_AXIS;
    ctx.font = '11px Inter, "Segoe UI", system-ui, sans-serif';
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';
    for (const tick of yTicks(series.maxDepth)) {
      ctx.fillText(formatDepthAmount(tick), layout.plotX - 8, depthToY(tick, layout));
    }
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    for (const px of xTicks(series.minPrice, series.maxPrice)) {
      ctx.fillText(formatDepthPrice(px), priceToX(px, layout), floorY + 6);
    }
  }, [series, size.h, size.w]);

  const onMove = useCallback(
    (event: React.MouseEvent<HTMLCanvasElement>) => {
      if (!series) {
        setHover(null);
        return;
      }
      const canvas = canvasRef.current;
      if (!canvas) return;
      const r = canvas.getBoundingClientRect();
      const layout = buildDepthLayout(series, r.width, r.height);
      setHover(hitTestDepth(event.clientX - r.left, event.clientY - r.top, series, layout));
    },
    [series],
  );

  return (
    <Panel data-testid="order-depth-chart">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('orderDepth.title')}
        </Text>
        {isFetching && !isLoading ? <Spin size="small" /> : null}
      </TitleRow>
      <Text variant="caption" color="secondary">
        {t('orderDepth.subtitle')}
      </Text>
      <MetricRow>
        <Text variant="caption" color="secondary">
          {t('orderDepth.metric')}
        </Text>
        <MetricChip
          type="button"
          $active={metric === 'base'}
          onClick={() => setMetric('base')}
        >
          {t('orderDepth.base')}
        </MetricChip>
        <MetricChip
          type="button"
          $active={metric === 'notional'}
          onClick={() => setMetric('notional')}
        >
          {t('orderDepth.notional')}
        </MetricChip>
      </MetricRow>

      {errorMessage ? (
        <Text variant="body" color="secondary">
          {errorMessage}
        </Text>
      ) : isLoading && !book ? (
        <Skeleton height={DEPTH_CHART_HEIGHT} />
      ) : !series ? (
        <Text variant="body" color="secondary">
          {t('orderDepth.empty')}
        </Text>
      ) : (
        <ChartFrame ref={frameRef}>
          <DepthCanvas
            data-testid="order-depth-canvas"
            ref={canvasRef}
            onMouseMove={onMove}
            onMouseLeave={() => setHover(null)}
          />
          {hover ? (
            <HoverCard $x={Math.min(hover.x + 12, size.w - 170)} $y={Math.max(8, hover.y - 56)}>
              <Text variant="caption" color="secondary">
                {hover.side === 'bid' ? t('orderDepth.bids') : t('orderDepth.asks')}
              </Text>
              <Text variant="numeric" color="primary">
                {formatDepthPrice(hover.price)}
              </Text>
              <Text variant="caption" color="secondary">
                {t('orderDepth.cumulative')}: {formatDepthAmount(hover.depth)}
              </Text>
            </HoverCard>
          ) : null}
        </ChartFrame>
      )}
    </Panel>
  );
}
