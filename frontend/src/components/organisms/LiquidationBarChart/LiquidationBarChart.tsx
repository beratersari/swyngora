import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  LIQ_BAR_INK,
  LIQ_BAR_LAST,
  LIQ_BAR_LONG,
  LIQ_BAR_PAD,
  LIQ_BAR_PLOT,
  LIQ_BAR_SHORT,
  LIQ_BAR_SPINE,
} from './LiquidationBarChart.constants';
import {
  isLevelsKind,
  maxSide,
  maxTotal,
  parseNotional,
  toLevelRows,
  toTimeRows,
} from './LiquidationBarChart.helpers';
import {
  ChartCanvas,
  HoverCard,
  LegendRow,
  MapFrame,
  Panel,
  Swatch,
  TipRow,
  TipTitle,
  TitleRow,
} from './LiquidationBarChart.styles';
import type { BarHover, LiquidationBarChartProps } from './LiquidationBarChart.types';

/**
 * CoinGlass-style liquidation map: price-level bars for one coin,
 * or time bars of total long/short when every coin is selected.
 */
export function LiquidationBarChart({
  data,
  isLoading,
  isFetching,
  errorMessage,
}: LiquidationBarChartProps) {
  const { t } = useTranslation('liquidations');
  const { formatCompact } = useDisplayCurrency();
  const frameRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [size, setSize] = useState({ w: 720, h: 460 });
  const [hover, setHover] = useState<BarHover | null>(null);
  const levels = useMemo(() => toLevelRows(data), [data]);
  const times = useMemo(() => toTimeRows(data), [data]);
  const mapMode = isLevelsKind(data);
  const lastPrice = parseNotional(data?.lastPrice);

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
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(size.w * dpr);
    canvas.height = Math.round(size.h * dpr);
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = LIQ_BAR_PLOT;
    ctx.fillRect(0, 0, size.w, size.h);
    if (mapMode) drawLevels(ctx, size.w, size.h, levels, lastPrice);
    else drawTotals(ctx, size.w, size.h, times);
  }, [lastPrice, levels, mapMode, size.h, size.w, times]);

  const onMove = useCallback(
    (event: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const r = canvas.getBoundingClientRect();
      const x = event.clientX - r.left;
      const y = event.clientY - r.top;
      const hit = mapMode
        ? hitLevel(x, y, r.width, r.height, levels, lastPrice)
        : hitTime(x, y, r.width, r.height, times);
      setHover(hit);
    },
    [lastPrice, levels, mapMode, times],
  );

  if (isLoading && !data) {
    return <Skeleton height={460} />;
  }
  if (errorMessage) {
    return <Alert type="error" showIcon message={errorMessage} />;
  }
  const empty = mapMode ? levels.every((row) => row.totalN <= 0) : times.every((row) => row.totalN <= 0);
  if (empty) {
    return <DeskEmpty title={t('chart.empty')} hint={t('chart.emptyHint')} />;
  }

  return (
    <Panel data-testid="liquidation-bar-chart">
      <TitleRow>
        <Text variant="h4" color="primary">
          {mapMode ? t('chart.levelsTitle') : t('chart.totalsTitle')}
        </Text>
        {isFetching && !isLoading ? <Spin size="small" /> : null}
      </TitleRow>
      <Text variant="caption" color="secondary">
        {mapMode ? t('chart.levelsHint') : t('chart.totalsHint')}
      </Text>
      <LegendRow>
        <Swatch $tone="long">{t('cards.long')}</Swatch>
        <Swatch $tone="short">{t('cards.short')}</Swatch>
        {mapMode ? <Swatch $tone="last">{t('chart.lastPrice')}</Swatch> : null}
      </LegendRow>
      <MapFrame ref={frameRef} role="img" aria-label={t('chart.aria')}>
        <ChartCanvas
          ref={canvasRef}
          data-testid="liquidation-bar-chart-canvas"
          onMouseMove={onMove}
          onMouseLeave={() => setHover(null)}
        />
        {hover ? (
          <HoverCard $x={hover.x} $y={hover.y} role="tooltip">
            <TipTitle>{hover.title}</TipTitle>
            <TipRow>
              <span>{t('cards.long')}</span>
              <span>{formatCompact(hover.longN, 'USDT')}</span>
            </TipRow>
            <TipRow>
              <span>{t('cards.short')}</span>
              <span>{formatCompact(hover.shortN, 'USDT')}</span>
            </TipRow>
            <TipRow>
              <span>{t('cards.total')}</span>
              <span>{formatCompact(hover.totalN, 'USDT')}</span>
            </TipRow>
          </HoverCard>
        ) : null}
      </MapFrame>
    </Panel>
  );
}

function plotBox(w: number, h: number) {
  const { left, right, top, bottom } = LIQ_BAR_PAD;
  return { x: left, y: top, w: Math.max(40, w - left - right), h: Math.max(40, h - top - bottom) };
}

function drawLevels(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  rows: ReturnType<typeof toLevelRows>,
  lastPrice: number,
) {
  const box = plotBox(w, h);
  const peak = maxSide(rows) || 1;
  const mid = box.x + box.w / 2;
  const rowH = box.h / Math.max(rows.length, 1);
  ctx.strokeStyle = LIQ_BAR_SPINE;
  ctx.beginPath();
  ctx.moveTo(mid, box.y);
  ctx.lineTo(mid, box.y + box.h);
  ctx.stroke();
  ctx.font = '11px Inter, Segoe UI, sans-serif';
  ctx.fillStyle = LIQ_BAR_INK;
  rows.forEach((row, i) => {
    const y = box.y + i * rowH + 1;
    const bh = Math.max(2, rowH - 2);
    const lw = (row.longN / peak) * (box.w / 2 - 4);
    const sw = (row.shortN / peak) * (box.w / 2 - 4);
    ctx.fillStyle = LIQ_BAR_LONG;
    ctx.fillRect(mid - lw, y, lw, bh);
    ctx.fillStyle = LIQ_BAR_SHORT;
    ctx.fillRect(mid, y, sw, bh);
    if (i % Math.ceil(rows.length / 8) === 0 || i === rows.length - 1) {
      ctx.fillStyle = LIQ_BAR_INK;
      ctx.textAlign = 'right';
      ctx.fillText(formatAxisPrice(row.price), box.x - 8, y + bh * 0.7);
    }
  });
  if (lastPrice > 0 && rows.length > 1) {
    const hi = rows[0]!.price;
    const lo = rows[rows.length - 1]!.price;
    const span = hi - lo || 1;
    const y = box.y + ((hi - lastPrice) / span) * box.h;
    ctx.strokeStyle = LIQ_BAR_LAST;
    ctx.setLineDash([4, 3]);
    ctx.beginPath();
    ctx.moveTo(box.x, y);
    ctx.lineTo(box.x + box.w, y);
    ctx.stroke();
    ctx.setLineDash([]);
  }
}

function drawTotals(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  rows: ReturnType<typeof toTimeRows>,
) {
  const box = plotBox(w, h);
  const peak = maxTotal(rows) || 1;
  const gap = 3;
  const bw = Math.max(3, box.w / Math.max(rows.length, 1) - gap);
  ctx.strokeStyle = '#E6EAF0';
  ctx.beginPath();
  ctx.moveTo(box.x, box.y + box.h);
  ctx.lineTo(box.x + box.w, box.y + box.h);
  ctx.stroke();
  ctx.font = '11px Inter, Segoe UI, sans-serif';
  rows.forEach((row, i) => {
    const x = box.x + i * (bw + gap);
    const longH = (row.longN / peak) * box.h;
    const shortH = (row.shortN / peak) * box.h;
    const yShort = box.y + box.h - shortH;
    const yLong = yShort - longH;
    ctx.fillStyle = LIQ_BAR_SHORT;
    ctx.fillRect(x, yShort, bw, shortH);
    ctx.fillStyle = LIQ_BAR_LONG;
    ctx.fillRect(x, yLong, bw, longH);
    if (i % Math.ceil(rows.length / 6) === 0) {
      ctx.fillStyle = LIQ_BAR_INK;
      ctx.textAlign = 'center';
      ctx.fillText(row.label, x + bw / 2, box.y + box.h + 16);
    }
  });
}

function hitLevel(
  x: number,
  y: number,
  w: number,
  h: number,
  rows: ReturnType<typeof toLevelRows>,
  lastPrice: number,
): BarHover | null {
  const box = plotBox(w, h);
  if (x < box.x || x > box.x + box.w || y < box.y || y > box.y + box.h || !rows.length) {
    return lastPrice ? null : null;
  }
  const i = Math.min(rows.length - 1, Math.max(0, Math.floor(((y - box.y) / box.h) * rows.length)));
  const row = rows[i];
  if (!row) return null;
  return {
    x: Math.min(x + 12, w - 190),
    y: Math.max(8, y - 72),
    title: formatAxisPrice(row.price),
    longN: row.longN,
    shortN: row.shortN,
    totalN: row.totalN,
  };
}

function hitTime(
  x: number,
  y: number,
  w: number,
  h: number,
  rows: ReturnType<typeof toTimeRows>,
): BarHover | null {
  const box = plotBox(w, h);
  if (x < box.x || x > box.x + box.w || y < box.y || y > box.y + box.h || !rows.length) {
    return null;
  }
  const i = Math.min(rows.length - 1, Math.max(0, Math.floor(((x - box.x) / box.w) * rows.length)));
  const row = rows[i];
  if (!row) return null;
  return {
    x: Math.min(x + 12, w - 190),
    y: Math.max(8, y - 72),
    title: row.label,
    longN: row.longN,
    shortN: row.shortN,
    totalN: row.totalN,
  };
}

function formatAxisPrice(n: number): string {
  if (!Number.isFinite(n)) return '—';
  if (n >= 1000) return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
  if (n >= 1) return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  return n.toLocaleString(undefined, { maximumFractionDigits: 6 });
}
