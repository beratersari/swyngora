import { useLayoutEffect, useMemo, useRef } from 'react';
import { createChart, LineSeries, type IChartApi, type ISeriesApi, type Time } from 'lightweight-charts';
import { financeColors } from '@/styles/tokens';
import { depthHistoryLine, depthPriceLine } from './helpers';
import { ChartBox } from './PerpLiquidityOverview.styles';

export function PerpLiquidityDepthChart({
  history,
}: {
  history?: { time?: string; total?: string; price?: string }[] | null;
}) {
  const boxRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const priceRef = useRef<ISeriesApi<'Line'> | null>(null);
  const line = useMemo(() => depthHistoryLine(history), [history]);
  const price = useMemo(() => depthPriceLine(history), [history]);
  const lineRef = useRef(line);
  const priceLineRef = useRef(price);
  lineRef.current = line;
  priceLineRef.current = price;
  const ready = line.length >= 2;

  useLayoutEffect(() => {
    if (!ready) return;
    const el = boxRef.current;
    if (!el) return;
    const chart = createChart(el, {
      width: Math.max(el.clientWidth || 320, 8),
      height: 120,
      layout: { background: { color: 'transparent' }, textColor: financeColors.silver },
      grid: {
        vertLines: { color: 'rgba(56, 97, 251, 0.08)' },
        horzLines: { color: 'rgba(56, 97, 251, 0.08)' },
      },
      rightPriceScale: { borderVisible: false },
      leftPriceScale: { visible: true, borderVisible: false },
      timeScale: { borderVisible: false, timeVisible: true, secondsVisible: false },
      handleScroll: false,
      handleScale: false,
    });
    const series = chart.addSeries(LineSeries, {
      color: financeColors.cobalt,
      lineWidth: 2,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    const priceSeries = chart.addSeries(LineSeries, {
      color: financeColors.warning,
      lineWidth: 2,
      priceScaleId: 'left',
      lastValueVisible: true,
      priceLineVisible: false,
    });
    chartRef.current = chart;
    seriesRef.current = series;
    priceRef.current = priceSeries;
    const initial = lineRef.current;
    if (initial.length > 0) {
      series.setData(initial.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    const initialPrice = priceLineRef.current;
    if (initialPrice.length > 0) {
      priceSeries.setData(initialPrice.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    chart.timeScale().fitContent();
    const ro = new ResizeObserver(() => {
      if (!boxRef.current || !chartRef.current) return;
      chartRef.current.applyOptions({ width: Math.max(boxRef.current.clientWidth || 320, 8), height: 120 });
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      priceRef.current = null;
    };
  }, [ready]);

  useLayoutEffect(() => {
    const series = seriesRef.current;
    const priceSeries = priceRef.current;
    const chart = chartRef.current;
    if (!series || !chart || !ready) return;
    series.setData(line.map((p) => ({ time: p.time as Time, value: p.value })));
    if (priceSeries) {
      priceSeries.setData(price.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    chart.timeScale().fitContent();
  }, [line, price, ready]);

  if (!ready) return null;
  return <ChartBox ref={boxRef} data-testid="perp-depth-history" />;
}
