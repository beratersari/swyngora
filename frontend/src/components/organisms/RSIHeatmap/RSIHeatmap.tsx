import { useCallback, useMemo, useRef, useState, type MouseEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useTheme } from 'styled-components';
import { RSI_DOT_RADIUS, RSI_HEAT_STOPS, RSI_HOVER_REACH } from './constants';
import {
  formatRSI,
  plotInner,
  plottedRows,
  plotX,
  plotY,
  rowKey,
  rowLabel,
  rsiFill,
  shouldLabelDot,
  tipOrigin,
} from './helpers';
import { Frame, Plot, Shell, Stats, Tip } from './RSIHeatmap.styles';
import type { RSIHeatmapProps, RSIHeatmapRow } from './RSIHeatmap.types';

export function RSIHeatmap({ data, isLoading, onOpen }: RSIHeatmapProps) {
  const { t } = useTranslation(['heatmap', 'common']);
  const theme = useTheme();
  const items = useMemo(() => data?.items ?? [], [data?.items]);
  const exchange = data?.exchange ?? 'binance';
  const frameElRef = useRef<HTMLDivElement | null>(null);
  const resizeRef = useRef<ResizeObserver | null>(null);
  const [size, setSize] = useState({ w: 960, h: 520 });
  const [hover, setHover] = useState<{ row: RSIHeatmapRow; x: number; y: number } | null>(null);

  const frameRef = useCallback((el: HTMLDivElement | null) => {
    resizeRef.current?.disconnect();
    resizeRef.current = null;
    frameElRef.current = el;
    if (!el) return;
    const apply = (entry?: ResizeObserverEntry) => {
      const w = Math.round(entry?.contentRect.width || el.clientWidth);
      const h = Math.round(entry?.contentRect.height || el.clientHeight);
      if (w > 0 && h > 0) {
        setSize((prev) => (prev.w === w && prev.h === h ? prev : { w, h }));
      }
    };
    apply();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver((entries) => apply(entries[0]));
    ro.observe(el);
    resizeRef.current = ro;
  }, []);

  const plotted = useMemo(() => plottedRows(items), [items]);
  const inner = plotInner(size.w, size.h);
  const avg = data?.averageRsi;
  const axis = theme.semantic.text.tertiary;
  const ink = theme.semantic.text.primary;
  const grid = theme.semantic.border.default;
  const bandLine = theme.semantic.text.secondary;

  const placeTip = useCallback((clientX: number, clientY: number, row: RSIHeatmapRow) => {
    const frame = frameElRef.current?.getBoundingClientRect();
    const tip = tipOrigin(
      frame ? clientX - frame.left : 16,
      frame ? clientY - frame.top : 16,
      frame?.width ?? size.w,
      frame?.height ?? size.h,
    );
    setHover({ row, x: tip.x, y: tip.y });
  }, [size.h, size.w]);

  const onFrameMove = useCallback((e: MouseEvent<HTMLDivElement>) => {
    const frame = frameElRef.current?.getBoundingClientRect();
    if (!frame) return;
    const tip = tipOrigin(e.clientX - frame.left, e.clientY - frame.top, frame.width, frame.height);
    setHover((cur) => (cur ? { ...cur, x: tip.x, y: tip.y } : cur));
  }, []);

  if (isLoading && items.length === 0) {
    return <Skeleton height={520} />;
  }
  if (items.length === 0) {
    return <DeskEmpty title={t('heatmap:empty')} />;
  }

  const ticks = [0, 30, 50, 70, 100];

  return (
    <Shell>
      <Stats>
        <span>
          {t('heatmap:rsi.average')} <strong>{formatRSI(avg)}</strong>
        </span>
        <span>
          {t('heatmap:rsi.oversold')} <strong>{data?.oversoldCount ?? 0}</strong>
        </span>
        <span>
          {t('heatmap:rsi.overbought')} <strong>{data?.overboughtCount ?? 0}</strong>
        </span>
        <span>
          {t('heatmap:rsi.coins')} <strong>{plotted.length}</strong>
        </span>
      </Stats>
      <Frame
        ref={frameRef}
        data-testid="rsi-heatmap"
        role="img"
        aria-label={t('heatmap:rsi.mapAria')}
        onMouseMove={onFrameMove}
        onMouseLeave={() => setHover(null)}
      >
        <Plot
          viewBox={`0 0 ${size.w} ${size.h}`}
          preserveAspectRatio="xMidYMid meet"
        >
          <defs>
            <linearGradient id="rsi-heat-scale" x1="0" y1="0" x2="0" y2="1">
              {[...RSI_HEAT_STOPS].reverse().map((stop) => (
                <stop key={stop.at} offset={`${100 - stop.at}%`} stopColor={stop.fill} />
              ))}
            </linearGradient>
          </defs>
          <g style={{ pointerEvents: 'none' }}>
          <rect
            data-testid="rsi-heat-scale"
            x={inner.x + inner.w + 10}
            y={inner.y}
            width={8}
            height={inner.h}
            fill="url(#rsi-heat-scale)"
            rx={2}
          />
          <rect x={inner.x} y={plotY(100, size.h)} width={inner.w} height={plotY(70, size.h) - plotY(100, size.h)} fill={theme.semantic.chart.down} opacity={0.08} />
          <rect x={inner.x} y={plotY(30, size.h)} width={inner.w} height={plotY(0, size.h) - plotY(30, size.h)} fill={theme.semantic.chart.up} opacity={0.08} />
          {ticks.map((tick) => {
            const y = plotY(tick, size.h);
            const strong = tick === 30 || tick === 70;
            return (
              <g key={tick}>
                <line
                  x1={inner.x}
                  x2={inner.x + inner.w}
                  y1={y}
                  y2={y}
                  stroke={strong ? bandLine : grid}
                  strokeDasharray={strong ? '4 4' : undefined}
                  strokeWidth={1}
                />
                <text x={inner.x - 8} y={y + 4} textAnchor="end" fill={axis} fontSize={11}>
                  {tick}
                </text>
              </g>
            );
          })}
          {avg != null ? (
            <line
              data-testid="rsi-avg-line"
              x1={inner.x}
              x2={inner.x + inner.w}
              y1={plotY(avg, size.h)}
              y2={plotY(avg, size.h)}
              stroke={theme.semantic.accent.default}
              strokeWidth={1.5}
            />
          ) : null}
          </g>
          {[...plotted]
            .sort((a, b) => {
              const ha = hover && rowKey(hover.row) === rowKey(a) ? 1 : 0;
              const hb = hover && rowKey(hover.row) === rowKey(b) ? 1 : 0;
              return ha - hb;
            })
            .map((row) => {
            const slot = plotted.indexOf(row) + 1;
            const cx = plotX(slot, plotted.length, size.w);
            const cy = plotY(row.rsi as number, size.h);
            const label = rowLabel(row);
            const active = hover != null && rowKey(hover.row) === rowKey(row);
            const r = active ? RSI_DOT_RADIUS + 2 : RSI_DOT_RADIUS;
            return (
              <g
                key={rowKey(row) || label}
                data-testid={`rsi-dot-${label}`}
                style={{ cursor: 'pointer' }}
                onMouseEnter={(e) => placeTip(e.clientX, e.clientY, row)}
                onMouseLeave={() => {
                  setHover((cur) => (cur && rowKey(cur.row) === rowKey(row) ? null : cur));
                }}
                onClick={() => row.symbol && onOpen?.(exchange, row.symbol)}
              >
                <circle cx={cx} cy={cy} r={RSI_HOVER_REACH} fill="transparent" />
                <circle
                  cx={cx}
                  cy={cy}
                  r={r}
                  fill={rsiFill(row.rsi, row.zone)}
                  stroke={active ? ink : theme.semantic.bg.canvas}
                  strokeWidth={active ? 1.5 : 1}
                />
                {shouldLabelDot(row, plotted.length) ? (
                  <text
                    x={cx}
                    y={cy - r - 5}
                    textAnchor="middle"
                    fill={ink}
                    fontSize={10}
                    fontWeight={600}
                    style={{ pointerEvents: 'none' }}
                  >
                    {label}
                  </text>
                ) : null}
              </g>
            );
          })}
          <g style={{ pointerEvents: 'none' }}>
          <text x={inner.x} y={size.h - 12} fill={axis} fontSize={11}>
            {t('heatmap:rsi.rankLeft')}
          </text>
          <text x={inner.x + inner.w} y={size.h - 12} textAnchor="end" fill={axis} fontSize={11}>
            {t('heatmap:rsi.rankRight')}
          </text>
          </g>
        </Plot>
        {hover ? (
          <Tip $x={hover.x} $y={hover.y}>
            <div>
              <strong>{rowLabel(hover.row)}</strong> · {formatRSI(hover.row.rsi)}
            </div>
            <div>
              {t('heatmap:rsi.rankN', { n: hover.row.rank })} · {hover.row.zone || '—'}
            </div>
          </Tip>
        ) : null}
      </Frame>
    </Shell>
  );
}
