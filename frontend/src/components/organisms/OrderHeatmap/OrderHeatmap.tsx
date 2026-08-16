import { Spin } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import {
  AXIS_INK,
  COB_RULE,
  DEFAULT_ORDER_HEATMAP_WINDOW,
  GRID_LINE,
  MID_STROKE,
  ORDER_HEATMAP_WINDOWS,
  PLOT_BG,
} from './OrderHeatmap.constants';
import {
  bloomIntensities,
  bucketsForColumn,
  buildLayout,
  depthColor,
  formatCollectedSpan,
  formatHeatNotional,
  formatHeatPrice,
  formatHeatTime,
  gradientStops,
  hitTest,
  intensityFromNotional,
  parseBookNumber,
  priceToY,
  xTickRects,
  yTicks,
} from './OrderHeatmap.helpers';
import {
  HeatCanvas,
  HoverCard,
  MapFrame,
  Panel,
  ScaleBar,
  ScaleLegend,
  TitleRow,
  WindowChip,
  WindowRow,
} from './OrderHeatmap.styles';
import type { HeatHover, OrderHeatmapProps } from './OrderHeatmap.types';

function windowLabel(seconds: number): string {
  if (seconds % 60 === 0) return `${seconds / 60}m`;
  return `${seconds}s`;
}

/**
 * Liquidity heatmap on the desk palette: green bids / red asks, history left,
 * wide current-book column on the right. One snapshot still fills the plot.
 */
export function OrderHeatmap({
  data,
  windowSeconds,
  onWindowChange,
  isLoading,
  isFetching,
  errorMessage,
}: OrderHeatmapProps) {
  const { t } = useTranslation('detail');
  const frameRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [size, setSize] = useState({ w: 720, h: 360 });
  const [hover, setHover] = useState<HeatHover | null>(null);
  const columns = data?.columns ?? [];
  const hasTape = columns.length > 0;

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
    ctx.imageSmoothingEnabled = true;
    ctx.imageSmoothingQuality = 'high';
    ctx.fillStyle = PLOT_BG;
    ctx.fillRect(0, 0, size.w, size.h);

    const layout = buildLayout(data, size.w, size.h);
    if (!layout) return;

    const step = layout.step > 0 ? layout.step : (layout.maxPrice - layout.minPrice) / 80;
    for (const rect of layout.rects) {
      const col = columns[rect.index];
      if (!col) continue;
      const buckets = bucketsForColumn(col, layout.prices, step);
      const heats = bloomIntensities(
        buckets.map((bucket) => {
          const heat = intensityFromNotional(bucket.notional, layout.peak);
          return bucket.isWall ? Math.min(1, heat + 0.1) : heat;
        }),
      );
      const grad = ctx.createLinearGradient(0, layout.plotY, 0, layout.plotY + layout.plotH);
      for (const stop of gradientStops(buckets, heats, layout, step)) {
        grad.addColorStop(stop.t, stop.color);
      }
      ctx.fillStyle = grad;
      ctx.fillRect(rect.x, layout.plotY, Math.max(1, rect.w), layout.plotH);

      if (rect.isCob) {
        for (let i = 0; i < buckets.length; i += 1) {
          const bucket = buckets[i];
          if (bucket.notional <= 0) continue;
          const yBottom = priceToY(bucket.price, layout);
          const yTop = priceToY(bucket.price + step, layout);
          const h = Math.max(1.5, yBottom - yTop);
          const barW = Math.max(3, (bucket.notional / Math.max(layout.peak, 1)) * rect.w * 0.88);
          ctx.fillStyle = depthColor(bucket.side, Math.min(1, (heats[i] ?? 0) + 0.12));
          ctx.fillRect(rect.x + 1, yTop, barW, h);
        }
        ctx.strokeStyle = COB_RULE;
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(rect.x + 0.5, layout.plotY);
        ctx.lineTo(rect.x + 0.5, layout.plotY + layout.plotH);
        ctx.stroke();
      }
    }

    ctx.strokeStyle = GRID_LINE;
    ctx.lineWidth = 1;
    for (const price of yTicks(layout)) {
      const y = priceToY(price, layout);
      ctx.beginPath();
      ctx.moveTo(layout.plotX, y);
      ctx.lineTo(layout.plotX + layout.plotW, y);
      ctx.stroke();
    }

    const lastMid = parseBookNumber(columns[columns.length - 1]?.mid);
    if (lastMid > 0) {
      const y = priceToY(lastMid, layout);
      ctx.strokeStyle = MID_STROKE;
      ctx.lineWidth = 1;
      ctx.setLineDash([4, 3]);
      ctx.beginPath();
      ctx.moveTo(layout.plotX, y);
      ctx.lineTo(layout.plotX + layout.plotW, y);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.fillStyle = MID_STROKE;
      ctx.font = '11px Inter, "Segoe UI", system-ui, sans-serif';
      ctx.textAlign = 'left';
      ctx.textBaseline = 'bottom';
      ctx.fillText(formatHeatPrice(lastMid, step), layout.plotX + 6, y - 3);
    }

    ctx.fillStyle = AXIS_INK;
    ctx.font = '11px Inter, "Segoe UI", system-ui, sans-serif';
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';
    for (const price of yTicks(layout)) {
      const y = priceToY(price, layout);
      ctx.fillText(formatHeatPrice(price, step), layout.plotX - 8, y);
    }
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    for (const rect of xTickRects(layout)) {
      const label = rect.isCob ? t('orderHeatmap.now') : formatHeatTime(rect.timeMs);
      ctx.fillText(label, rect.x + rect.w / 2, layout.plotY + layout.plotH + 6);
    }

    const scale = ctx.createLinearGradient(0, layout.plotY + layout.plotH, 0, layout.plotY);
    scale.addColorStop(0, depthColor('bid', 1));
    scale.addColorStop(0.5, PLOT_BG);
    scale.addColorStop(1, depthColor('ask', 1));
    ctx.fillStyle = scale;
    ctx.fillRect(layout.scaleX, layout.plotY, layout.scaleW, layout.plotH);
  }, [columns, data, size.h, size.w, t]);

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
      setHover(hitTest(event.clientX - r.left, event.clientY - r.top, data, layout));
    },
    [data],
  );

  const activeWindow = windowSeconds || data?.windowSeconds || DEFAULT_ORDER_HEATMAP_WINDOW;
  const layout = hasTape ? buildLayout(data, size.w, size.h) : null;
  const collected = formatCollectedSpan(data?.from, data?.to);

  return (
    <Panel data-testid="order-heatmap">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('orderHeatmap.title')}
        </Text>
        {isFetching && !isLoading ? <Spin size="small" /> : null}
      </TitleRow>
      <Text variant="caption" color="secondary">
        {t('orderHeatmap.subtitle')}
      </Text>
      <WindowRow>
        <Text variant="caption" color="secondary">
          {t('orderHeatmap.window')}
        </Text>
        {ORDER_HEATMAP_WINDOWS.map((sec) => (
          <WindowChip
            key={sec}
            type="button"
            $active={sec === activeWindow}
            onClick={() => onWindowChange(sec)}
          >
            {windowLabel(sec)}
          </WindowChip>
        ))}
        {collected ? (
          <Text variant="caption" color="secondary" data-testid="order-heatmap-span">
            {t('orderHeatmap.collected', { span: collected })}
          </Text>
        ) : null}
        <ScaleLegend>
          <Text variant="caption" color="secondary">
            {t('orderHeatmap.low')}
          </Text>
          <ScaleBar />
          <Text variant="caption" color="secondary">
            {t('orderHeatmap.high')}
          </Text>
        </ScaleLegend>
      </WindowRow>

      {errorMessage ? (
        <Text variant="body" color="secondary">
          {errorMessage}
        </Text>
      ) : isLoading && !hasTape ? (
        <Skeleton height={320} />
      ) : !hasTape ? (
        <Text variant="body" color="secondary">
          {t('orderHeatmap.empty')}
        </Text>
      ) : (
        <MapFrame ref={frameRef}>
          <HeatCanvas
            ref={canvasRef}
            data-testid="order-heatmap-canvas"
            onMouseMove={onMove}
            onMouseLeave={() => setHover(null)}
          />
          {hover ? (
            <HoverCard
              $x={Math.min(hover.x + 12, size.w - 190)}
              $y={Math.max(8, hover.y - 80)}
              data-testid="order-heatmap-hover"
            >
              <Text variant="caption" color="secondary">
                {formatHeatTime(hover.timeMs)} · {formatHeatPrice(hover.price, layout?.step ?? 0)}
              </Text>
              <Text variant="caption" color="primary">
                {t('orderHeatmap.size')}: {formatHeatNotional(hover.bid || hover.ask)}
                {hover.bidWall || hover.askWall ? ` · ${t('orderBook.wall')}` : ''}
              </Text>
              {hover.mid ? (
                <Text variant="caption" color="secondary">
                  {t('orderBook.mid')}: {formatHeatPrice(hover.mid, layout?.step ?? 0)}
                </Text>
              ) : null}
            </HoverCard>
          ) : null}
        </MapFrame>
      )}
    </Panel>
  );
}
