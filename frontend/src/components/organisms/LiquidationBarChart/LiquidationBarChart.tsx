import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Select, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  LIQ_BAR_GRID,
  LIQ_BAR_INK,
  LIQ_BAR_LAST,
  LIQ_BAR_LINE,
  LIQ_BAR_LONG,
  LIQ_BAR_PAD,
  LIQ_BAR_PLOT,
  LIQ_BAR_SHORT,
  LIQ_BAR_SPINE,
  LIQ_LEVERAGE_COLOR,
  LIQ_LEVERAGE_ORDER,
} from './LiquidationBarChart.constants';
import {
  barSide,
  cumSliceNotional,
  formatAxisNotional,
  formatAxisPrice,
  formatHitRate,
  frozenLevelsView,
  isLevelsKind,
  maxCum,
  maxTotal,
  newestLevelSamples,
  legendOpacity,
  parseNotional,
  pickLevelsReview,
  sliceConfidence,
  toLevelRows,
  toTimeRows,
} from './LiquidationBarChart.helpers';
import {
  ChartCanvas,
  HoverCard,
  LegendRow,
  MapFrame,
  Panel,
  Reliability,
  ReviewTable,
  SignalScroll,
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
  const [asOf, setAsOf] = useState('live');
  const review = pickLevelsReview(data);
  const reviewHorizons = review?.horizons ?? [];
  const reviewBuckets = review?.byLeverage ?? [];
  const reviewSamples = newestLevelSamples(review);
  const frozen = useMemo(() => frozenLevelsView(data, reviewSamples, asOf), [data, reviewSamples, asOf]);
  const levels = useMemo(() => toLevelRows(frozen), [frozen]);
  const times = useMemo(() => toTimeRows(data), [data]);
  const mapMode = isLevelsKind(data);
  const lastPrice = parseNotional(frozen?.lastPrice);
  const viewingPast = asOf !== 'live' && (frozen?.levels?.length ?? 0) > 0;

  useEffect(() => {
    setAsOf('live');
  }, [data?.symbol, data?.exchange]);

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
  const empty = mapMode
    ? levels.every((row) => row.totalN <= 0 && row.cumTotal <= 0)
    : times.every((row) => row.totalN <= 0);
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
      {mapMode && reviewSamples.some((s) => (s.levels?.length ?? 0) > 0) ? (
        <Select
          aria-label={t('chart.review.asOfAria')}
          value={reviewSamples.some((s) => s.time === asOf) ? asOf : 'live'}
          style={{ maxWidth: 420 }}
          onChange={setAsOf}
          options={[
            { value: 'live', label: t('chart.review.asOfLive') },
            ...reviewSamples
              .filter((s) => (s.levels?.length ?? 0) > 0)
              .map((s) => ({
                value: s.time ?? '',
                label: t('chart.review.asOfEstimate', {
                  time: formatSampleTime(s.time),
                  last: formatAxisPrice(parseNotional(s.lastPrice)),
                  lean:
                    s.lean === 'short'
                      ? t('chart.review.leanShort')
                      : s.lean === 'long'
                        ? t('chart.review.leanLong')
                        : t('chart.review.leanEven'),
                  blend:
                    s.blend === 'top_positions'
                      ? t('chart.review.blendTop')
                      : t('chart.review.blendCrowd'),
                }),
              })),
          ]}
        />
      ) : null}
      <Text variant="caption" color="secondary">
        {viewingPast ? t('chart.review.frozenHint') : mapMode ? t('chart.levelsHint') : t('chart.totalsHint')}
      </Text>
      {data?.missing && data.missing.length > 0 ? (
        <Alert
          type="warning"
          showIcon
          message={t('feed.missingVenues', { venues: data.missing.join(', ') })}
        />
      ) : null}
      <LegendRow>
        {mapMode
          ? LIQ_LEVERAGE_ORDER.map((lev) => (
              <Swatch
                key={lev}
                $color={LIQ_LEVERAGE_COLOR[lev]}
                style={{ opacity: legendOpacity(reviewBuckets, lev) }}
              >
                {t('chart.leverage', { lev })}
              </Swatch>
            ))
          : (
            <>
              <Swatch $tone="long">{t('cards.long')}</Swatch>
              <Swatch $tone="short">{t('cards.short')}</Swatch>
            </>
          )}
        {mapMode ? <Swatch $tone="last">{t('chart.lastPrice')}</Swatch> : null}
        {mapMode ? <Swatch $tone="line">{t('chart.cumLine')}</Swatch> : null}
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
            {hover.cumTotal != null && hover.cumTotal > 0 ? (
              <TipRow>
                <span>{t('chart.cumToLevel')}</span>
                <span>{formatCompact(hover.cumTotal, 'USDT')}</span>
              </TipRow>
            ) : null}
            {hover.cumLong != null && hover.cumLong > 0 ? (
              <TipRow>
                <span>{t('chart.cumLong')}</span>
                <span>{formatCompact(hover.cumLong, 'USDT')}</span>
              </TipRow>
            ) : null}
            {hover.cumShort != null && hover.cumShort > 0 ? (
              <TipRow>
                <span>{t('chart.cumShort')}</span>
                <span>{formatCompact(hover.cumShort, 'USDT')}</span>
              </TipRow>
            ) : null}
            {hover.byLeverage?.map((sl) => (
              <TipRow key={sl.leverage}>
                <span>{t('chart.leverage', { lev: sl.leverage })}</span>
                <span>{formatCompact(sl.longN + sl.shortN, 'USDT')}</span>
              </TipRow>
            ))}
          </HoverCard>
        ) : null}
      </MapFrame>
      {mapMode && (reviewHorizons.length > 0 || reviewBuckets.length > 0) ? (
        <>
          <Text variant="h4" color="primary">
            {t('chart.review.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {data?.review?.summary || t('chart.review.hint')}
          </Text>
          {reviewHorizons.length > 0 ? (
            <ReviewTable data-testid="liquidation-levels-review">
              <thead>
                <tr>
                  <th>{t('chart.review.horizon')}</th>
                  <th>{t('chart.review.signals')}</th>
                  <th>{t('chart.review.validated')}</th>
                  <th>{t('chart.review.pending')}</th>
                  <th>{t('chart.review.hits')}</th>
                  <th>{t('chart.review.hitRate')}</th>
                  <th>{t('chart.review.fill')}</th>
                </tr>
              </thead>
              <tbody>
                {reviewHorizons.map((row) => (
                  <tr key={row.horizon ?? 'h'}>
                    <td>{row.horizon ?? '—'}</td>
                    <td>{row.signals ?? 0}</td>
                    <td>{row.validated ?? 0}</td>
                    <td>{row.pending ?? 0}</td>
                    <td>{row.hits ?? 0}</td>
                    <td>{row.validated ? formatHitRate(row.hitRate) : '—'}</td>
                    <td>{row.validated ? formatHitRate(row.fillRate) : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </ReviewTable>
          ) : null}
          {reviewBuckets.length > 0 ? (
            <>
              <Text variant="h4" color="primary">
                {t('chart.review.reliabilityTitle')}
              </Text>
              <Text variant="caption" color="secondary">
                {t('chart.review.reliabilityHint')}
              </Text>
              <ReviewTable data-testid="liquidation-levels-reliability">
                <thead>
                  <tr>
                    <th>{t('chart.review.level')}</th>
                    <th>{t('chart.review.side')}</th>
                    <th>{t('chart.review.validated')}</th>
                    <th>{t('chart.review.hitRate')}</th>
                    <th>{t('chart.review.fill')}</th>
                    <th>{t('chart.review.reliability')}</th>
                    <th>{t('chart.review.confidence')}</th>
                    <th>{t('chart.review.amountKept')}</th>
                  </tr>
                </thead>
                <tbody>
                  {reviewBuckets.map((row) => (
                    <tr key={`${row.leverage}-${row.side}`}>
                      <td>{t('chart.leverage', { lev: row.leverage })}</td>
                      <td>
                        {row.side === 'long' ? t('cards.long') : t('cards.short')}
                      </td>
                      <td>{row.validated ?? 0}</td>
                      <td>{row.validated ? formatHitRate(row.hitRate) : '—'}</td>
                      <td>{row.validated ? formatHitRate(row.fillRate) : '—'}</td>
                      <td>
                        <Reliability $tone={row.reliability ?? 'unknown'}>
                          {t(`chart.review.grades.${row.reliability ?? 'unknown'}`)}
                        </Reliability>
                      </td>
                      <td>{row.confidence ? formatHitRate(row.confidence) : '—'}</td>
                      <td>{row.sizeScale != null && row.sizeScale > 0 ? formatHitRate(row.sizeScale) : '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </ReviewTable>
            </>
          ) : null}
          {reviewSamples.length > 0 ? (
            <>
              <Text variant="h4" color="primary">
                {t('chart.review.samplesTitle')}
              </Text>
              <Text variant="caption" color="secondary">
                {t('chart.review.samplesHint')}
              </Text>
              <SignalScroll>
                <ReviewTable data-testid="liquidation-levels-samples">
                  <thead>
                    <tr>
                      <th>{t('chart.review.when')}</th>
                      <th>{t('chart.review.last')}</th>
                      <th>{t('chart.review.level')}</th>
                      <th>{t('chart.review.side')}</th>
                      <th>{t('chart.review.predicted')}</th>
                      <th>{t('chart.review.realized')}</th>
                      <th>{t('chart.review.result')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {reviewSamples.flatMap((sample) =>
                      (sample.bands ?? []).map((band, i) => (
                        <tr key={`${sample.time}-${band.leverage}-${band.side}-${i}`}>
                          <td>{formatSampleTime(sample.time)}</td>
                          <td>{formatAxisPrice(parseNotional(sample.lastPrice))}</td>
                          <td>{t('chart.leverage', { lev: band.leverage })}</td>
                          <td>
                            {band.side === 'long' ? t('cards.long') : t('cards.short')}
                          </td>
                          <td>{formatCompact(parseNotional(band.predicted), 'USDT')}</td>
                          <td>{formatCompact(parseNotional(band.realized), 'USDT')}</td>
                          <td>{t(`chart.review.status.${band.status ?? 'unknown'}`)}</td>
                        </tr>
                      )),
                    )}
                  </tbody>
                </ReviewTable>
              </SignalScroll>
            </>
          ) : null}
        </>
      ) : null}
    </Panel>
  );
}

function formatSampleTime(raw: string | undefined): string {
  const t = Date.parse(raw ?? '');
  if (!Number.isFinite(t)) return '—';
  return new Date(t).toISOString().slice(5, 16).replace('T', ' ');
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
  const peak = maxCum(rows) || 1;
  const rowH = box.h / Math.max(rows.length, 1);
  const plotW = box.w - 8;
  ctx.font = '11px Inter, Segoe UI, sans-serif';

  const xTicks = 4;
  for (let t = 1; t <= xTicks; t += 1) {
    const x = box.x + (t / xTicks) * plotW;
    ctx.strokeStyle = LIQ_BAR_GRID;
    ctx.beginPath();
    ctx.moveTo(x, box.y);
    ctx.lineTo(x, box.y + box.h);
    ctx.stroke();
    ctx.fillStyle = LIQ_BAR_INK;
    ctx.textAlign = 'center';
    ctx.fillText(formatAxisNotional((t / xTicks) * peak), x, box.y + box.h + 16);
  }

  const priceEvery = Math.max(1, Math.ceil(rows.length / 8));
  rows.forEach((row, i) => {
    const y = box.y + i * rowH;
    const bh = Math.max(2, rowH - 1);
    if (i % priceEvery === 0 || i === rows.length - 1) {
      ctx.strokeStyle = LIQ_BAR_GRID;
      ctx.beginPath();
      ctx.moveTo(box.x, y);
      ctx.lineTo(box.x + box.w, y);
      ctx.stroke();
      ctx.fillStyle = LIQ_BAR_INK;
      ctx.textAlign = 'right';
      ctx.fillText(formatAxisPrice(row.price), box.x - 8, y + bh * 0.75);
    }
  });

  ctx.strokeStyle = LIQ_BAR_SPINE;
  ctx.beginPath();
  ctx.moveTo(box.x, box.y);
  ctx.lineTo(box.x, box.y + box.h);
  ctx.lineTo(box.x + box.w, box.y + box.h);
  ctx.stroke();

  const tips: { x: number; y: number }[] = [];
  rows.forEach((row, i) => {
    const y = box.y + i * rowH + 0.5;
    const bh = Math.max(2, rowH - 1);
    const side = barSide(row, lastPrice);
    let x = box.x + 1;
    for (const lev of LIQ_LEVERAGE_ORDER) {
      const n = cumSliceNotional(row, lev, side);
      if (n <= 0) continue;
      const conf = sliceConfidence(row, lev, side);
      const bw = (n / peak) * plotW;
      ctx.globalAlpha = 0.28 + 0.72 * conf;
      ctx.fillStyle = LIQ_LEVERAGE_COLOR[lev];
      ctx.fillRect(x, y, Math.max(1, bw), bh);
      x += bw;
    }
    ctx.globalAlpha = 1;
    const tipX = box.x + 1 + (row.cumTotal / peak) * plotW;
    tips.push({ x: tipX, y: y + bh / 2 });
  });

  if (tips.length > 1) {
    ctx.strokeStyle = LIQ_BAR_LINE;
    ctx.lineWidth = 1.75;
    ctx.beginPath();
    tips.forEach((p, i) => {
      if (i === 0) ctx.moveTo(p.x, p.y);
      else ctx.lineTo(p.x, p.y);
    });
    ctx.stroke();
    ctx.lineWidth = 1;
  }

  if (lastPrice > 0 && rows.length > 1) {
    const hi = rows[0]!.price;
    const lo = rows[rows.length - 1]!.price;
    const span = hi - lo || 1;
    const y = box.y + ((hi - lastPrice) / span) * box.h;
    ctx.strokeStyle = LIQ_BAR_LAST;
    ctx.setLineDash([5, 4]);
    ctx.beginPath();
    ctx.moveTo(box.x, y);
    ctx.lineTo(box.x + box.w, y);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = LIQ_BAR_LAST;
    ctx.textAlign = 'left';
    ctx.fillText(formatAxisPrice(lastPrice), box.x + 6, Math.max(box.y + 12, y - 4));
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
  ctx.strokeStyle = LIQ_BAR_SPINE;
  ctx.beginPath();
  ctx.moveTo(box.x, box.y + box.h);
  ctx.lineTo(box.x + box.w, box.y + box.h);
  ctx.stroke();
  ctx.font = '11px Inter, Segoe UI, sans-serif';
  const tips: { x: number; y: number }[] = [];
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
    tips.push({ x: x + bw / 2, y: yLong });
    if (i % Math.ceil(rows.length / 6) === 0) {
      ctx.fillStyle = LIQ_BAR_INK;
      ctx.textAlign = 'center';
      ctx.fillText(row.label, x + bw / 2, box.y + box.h + 16);
    }
  });
  if (tips.length > 1) {
    ctx.strokeStyle = LIQ_BAR_LINE;
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    tips.forEach((p, i) => {
      if (i === 0) ctx.moveTo(p.x, p.y);
      else ctx.lineTo(p.x, p.y);
    });
    ctx.stroke();
    ctx.lineWidth = 1;
  }
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
    x: Math.min(x + 12, w - 220),
    y: Math.max(8, y - 96),
    title: formatAxisPrice(row.price),
    longN: row.longN,
    shortN: row.shortN,
    totalN: row.totalN,
    cumLong: row.cumLong,
    cumShort: row.cumShort,
    cumTotal: row.cumTotal,
    byLeverage: row.byLeverage,
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


