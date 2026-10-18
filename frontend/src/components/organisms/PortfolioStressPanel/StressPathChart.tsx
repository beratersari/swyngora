import { useEffect, useLayoutEffect, useMemo, useRef } from 'react';
import { createChart, LineSeries, type IChartApi, type ISeriesApi, type UTCTimestamp } from 'lightweight-charts';
import { Text } from '@/components/atoms/Text';
import { semanticColors } from '@/styles/tokens';

type PathPoint = { at?: string; equity?: number };

type StressPathChartProps = {
  points: PathPoint[];
  title: string;
  hint: string;
  low?: string;
};

type LinePoint = { time: UTCTimestamp; value: number };

function toLine(points: PathPoint[]): LinePoint[] {
  const rows: LinePoint[] = [];
  for (const point of points) {
    const ts = point.at ? Math.floor(Date.parse(point.at) / 1000) : NaN;
    if (!Number.isFinite(ts) || point.equity == null || !Number.isFinite(point.equity)) continue;
    rows.push({ time: ts as UTCTimestamp, value: point.equity });
  }
  rows.sort((a, b) => a.time - b.time);
  const deduped: LinePoint[] = [];
  for (const row of rows) {
    const prev = deduped[deduped.length - 1];
    if (prev && prev.time === row.time) {
      deduped[deduped.length - 1] = row;
      continue;
    }
    deduped.push(row);
  }
  return deduped;
}

export function StressPathChart({ points, title, hint, low }: StressPathChartProps) {
  const boxRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const line = useMemo(() => toLine(points), [points]);
  const lineRef = useRef(line);
  lineRef.current = line;
  const endedDown = line.length >= 2 && line[line.length - 1].value < line[0].value;

  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el || lineRef.current.length < 2) return;
    const chart = createChart(el, {
      width: Math.max(el.clientWidth || 320, 8),
      height: 180,
      layout: { background: { color: 'transparent' }, textColor: semanticColors.text.secondary },
      grid: {
        vertLines: { color: 'rgba(79, 212, 165, 0.08)' },
        horzLines: { color: 'rgba(79, 212, 165, 0.08)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: { borderVisible: false, secondsVisible: false },
      handleScroll: false,
      handleScale: false,
    });
    const series = chart.addSeries(LineSeries, {
      color: endedDown ? semanticColors.chart.down : semanticColors.chart.up,
      lineWidth: 2,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    series.setData(lineRef.current);
    chart.timeScale().fitContent();
    chartRef.current = chart;
    seriesRef.current = series;
    const ro = new ResizeObserver(() => {
      const host = boxRef.current;
      const live = chartRef.current;
      if (!host || !live) return;
      const width = host.clientWidth || 0;
      if (width < 8) return;
      live.applyOptions({ width, height: 180 });
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [endedDown, line]);

  useEffect(() => {
    const series = seriesRef.current;
    const chart = chartRef.current;
    if (!series || !chart || line.length < 2) return;
    series.setData(line);
    chart.timeScale().fitContent();
  }, [line]);

  if (line.length < 2) return null;

  return (
    <div data-testid="portfolio-stress-path">
      <Text variant="body" color="primary">
        {title}
      </Text>
      <Text variant="caption" color="secondary">
        {hint}
      </Text>
      {low ? (
        <Text variant="caption" color="secondary">
          {low}
        </Text>
      ) : null}
      <div ref={boxRef} style={{ height: 180, width: '100%' }} />
    </div>
  );
}
