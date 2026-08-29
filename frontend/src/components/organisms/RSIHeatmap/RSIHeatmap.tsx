import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useTheme } from 'styled-components';
import { RSI_DOT_RADIUS } from './constants';
import {
  formatRSI,
  nearestDot,
  plotInner,
  plotX,
  plotY,
  rowLabel,
  rsiFill,
  shouldLabelDot,
} from './helpers';
import { Frame, Plot, Shell, Stats, Tip } from './RSIHeatmap.styles';
import type { RSIHeatmapProps, RSIHeatmapRow } from './RSIHeatmap.types';

export function RSIHeatmap({ data, isLoading, onOpen }: RSIHeatmapProps) {
  const { t } = useTranslation(['heatmap', 'common']);
  const theme = useTheme();
  const items = useMemo(() => data?.items ?? [], [data?.items]);
  const exchange = data?.exchange ?? 'binance';
  const frameRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 960, h: 520 });
  const [hover, setHover] = useState<{ row: RSIHeatmapRow; x: number; y: number } | null>(null);

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

  const plotted = useMemo(() => items.filter((row) => row.rsi != null), [items]);
  const inner = plotInner(size.w, size.h);
  const avg = data?.averageRsi;
  const axis = theme.semantic.text.tertiary;
  const ink = theme.semantic.text.primary;
  const grid = theme.semantic.border.default;
  const bandLine = theme.semantic.text.secondary;

  const onMove = useCallback(
    (clientX: number, clientY: number) => {
      const el = frameRef.current!;
      const r = el.getBoundingClientRect();
      const mx = clientX - r.left;
      const my = clientY - r.top;
      const row = nearestDot(plotted, mx, my, size.w, size.h);
      if (!row) {
        setHover(null);
        return;
      }
      setHover({
        row,
        x: Math.min(mx + 14, size.w - 170),
        y: Math.min(my + 14, size.h - 88),
      });
    },
    [plotted, size.h, size.w],
  );

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
        onMouseLeave={() => setHover(null)}
      >
        <Plot viewBox={`0 0 ${size.w} ${size.h}`} onMouseMove={(e) => onMove(e.clientX, e.clientY)}>
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
          {plotted.map((row) => {
            const cx = plotX(row.rank ?? 0, plotted.length, size.w);
            const cy = plotY(row.rsi as number, size.h);
            const label = rowLabel(row);
            const r = hover?.row.symbol === row.symbol ? RSI_DOT_RADIUS + 1.5 : RSI_DOT_RADIUS;
            return (
              <g
                key={row.symbol ?? label}
                data-testid={`rsi-dot-${label}`}
                style={{ cursor: 'pointer' }}
                onClick={() => row.symbol && onOpen?.(exchange, row.symbol)}
              >
                <circle
                  cx={cx}
                  cy={cy}
                  r={r}
                  fill={rsiFill(row.zone)}
                  stroke={theme.semantic.bg.canvas}
                  strokeWidth={1}
                />
                {shouldLabelDot(row, plotted.length) ? (
                  <text x={cx} y={cy - r - 5} textAnchor="middle" fill={ink} fontSize={10} fontWeight={600}>
                    {label}
                  </text>
                ) : null}
              </g>
            );
          })}
          <text x={inner.x} y={size.h - 12} fill={axis} fontSize={11}>
            {t('heatmap:rsi.rankLeft')}
          </text>
          <text x={inner.x + inner.w} y={size.h - 12} textAnchor="end" fill={axis} fontSize={11}>
            {t('heatmap:rsi.rankRight')}
          </text>
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
