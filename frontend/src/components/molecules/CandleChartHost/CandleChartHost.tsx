import { useEffect, useLayoutEffect, useRef } from 'react';
import {
  CandlestickSeries,
  CrosshairMode,
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
} from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { palette, semanticColors } from '@/styles/tokens';
import type { CandleChartHostProps } from './CandleChartHost.types';
import { DEFAULT_HEIGHT } from './CandleChartHost.constants';
import {
  ChartContainer,
  ChartShell,
  ChartSkeletonLayer,
} from './CandleChartHost.styles';
import {
  candleDataSignature,
  chartPriceFormatFromCandles,
  overlaysSignature,
  toCandlestickData,
  toLineData,
} from './helpers';

/**
 * Thin host for TradingView Lightweight Charts candlesticks.
 * Optional line overlays (EMA) share the price scale.
 *
 * Chart DOM stays mounted across loading and overlay toggles so the canvas
 * is not torn down (avoids intermittent blank charts when toggling EMA).
 * Series updates are no-ops unless candle/overlay signatures change, so
 * React re-renders do not fight the crosshair or mouse tracking.
 */
export function CandleChartHost({
  data,
  overlays = [],
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
}: CandleChartHostProps) {
  const { t } = useTranslation('common');
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());
  const lastCandleSigRef = useRef<string>('');
  const lastOverlaySigRef = useRef<string>('');
  const hasFittedRef = useRef(false);
  const lastPriceFormatKeyRef = useRef<string>('');

  // Create chart once (layout phase so the data effect below can sync same frame).
  useLayoutEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const chart = createChart(el, {
      height,
      // Let the library own ResizeObserver — avoids width thrash from manual RO.
      autoSize: true,
      layout: {
        background: { color: palette.richBlack },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: semanticColors.chart.grid },
        horzLines: { color: semanticColors.chart.grid },
      },
      rightPriceScale: {
        borderColor: semanticColors.border.default,
        // Reduce scale-width oscillation when crosshair labels appear/disappear.
        minimumWidth: 72,
      },
      timeScale: {
        borderColor: semanticColors.border.default,
      },
      // Free-tracking dashed crosshair (Magnet snaps to bar OHLC and feels jumpy).
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: {
          color: semanticColors.border.strong,
          width: 1,
          style: 3, // LineStyle.LargeDashed
          labelBackgroundColor: semanticColors.bg.elevated,
        },
        horzLine: {
          color: semanticColors.border.strong,
          width: 1,
          style: 3, // LineStyle.LargeDashed
          labelBackgroundColor: semanticColors.bg.elevated,
        },
      },
      handleScroll: {
        mouseWheel: true,
        pressedMouseMove: true,
        horzTouchDrag: true,
        vertTouchDrag: true,
      },
      handleScale: {
        axisPressedMouseMove: true,
        mouseWheel: true,
        pinch: true,
      },
    });

    const series = chart.addSeries(CandlestickSeries, {
      upColor: semanticColors.chart.up,
      downColor: semanticColors.chart.down,
      borderVisible: false,
      wickUpColor: semanticColors.chart.up,
      wickDownColor: semanticColors.chart.down,
      // Default is precision: 2 — overwritten from candle magnitudes on data load.
      priceFormat: { type: 'price', precision: 2, minMove: 0.01 },
    });

    chartRef.current = chart;
    seriesRef.current = series;

    return () => {
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      overlayRefs.current.clear();
      lastCandleSigRef.current = '';
      lastOverlaySigRef.current = '';
      hasFittedRef.current = false;
      lastPriceFormatKeyRef.current = '';
    };
    // Intentionally mount-once: height is applied via a separate effect.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- chart lifecycle is mount/unmount only
  }, []);

  useEffect(() => {
    chartRef.current?.applyOptions({ height });
  }, [height]);

  // Sync series only when content actually changes — never on pure React re-renders.
  useLayoutEffect(() => {
    const chart = chartRef.current;
    const candleSeries = seriesRef.current;
    if (!chart || !candleSeries) return;

    const candleSig = candleDataSignature(data);
    const overlaySig = overlaysSignature(overlays);
    const candlesChanged = candleSig !== lastCandleSigRef.current;
    const overlaysChanged = overlaySig !== lastOverlaySigRef.current;

    // Critical: bail out so we never call fitContent / setVisibleLogicalRange /
    // setData while the user is moving the crosshair on a re-render that only
    // brought new array identities with the same series content.
    if (!candlesChanged && !overlaysChanged) {
      return;
    }

    const priceFormat = chartPriceFormatFromCandles(data);
    const priceFormatKey = `${priceFormat.precision}:${priceFormat.minMove}`;
    const priceFormatChanged = priceFormatKey !== lastPriceFormatKeyRef.current;

    if (candlesChanged) {
      candleSeries.setData(toCandlestickData(data));
      lastCandleSigRef.current = candleSig;
    }

    if (candlesChanged || priceFormatChanged) {
      candleSeries.applyOptions({ priceFormat });
      // Wider axis for long micro-price labels (e.g. 0.00000123).
      const labelWidth = Math.min(140, Math.max(72, 48 + priceFormat.precision * 7));
      chart.applyOptions({
        rightPriceScale: { minimumWidth: labelWidth },
      });
      lastPriceFormatKeyRef.current = priceFormatKey;
    }

    if (overlaysChanged || priceFormatChanged) {
      const nextIds = new Set(overlays.map((o) => o.id));

      for (const [id, lineSeries] of overlayRefs.current) {
        if (!nextIds.has(id)) {
          chart.removeSeries(lineSeries);
          overlayRefs.current.delete(id);
        }
      }

      for (const overlay of overlays) {
        let line = overlayRefs.current.get(overlay.id);
        if (!line) {
          line = chart.addSeries(LineSeries, {
            color: overlay.color,
            lineWidth: 2,
            title: overlay.title ?? overlay.id,
            priceLineVisible: false,
            // Avoid last-value labels resizing the price scale during hover.
            lastValueVisible: false,
            priceFormat,
          });
          overlayRefs.current.set(overlay.id, line);
        } else {
          line.applyOptions({
            color: overlay.color,
            title: overlay.title ?? overlay.id,
            priceFormat,
          });
        }
        if (overlaysChanged) {
          line.setData(toLineData(overlay.data));
        }
      }

      if (overlaysChanged) {
        lastOverlaySigRef.current = overlaySig;
      }
    }

    // Only re-fit when candle bars change (or first paint). Overlay toggles
    // leave the time scale alone so zoom/pan/crosshair stay stable.
    if (data.length > 0 && (candlesChanged || !hasFittedRef.current)) {
      chart.timeScale().fitContent();
      hasFittedRef.current = true;
    }
  }, [data, overlays]);

  const showSkeleton = isLoading && data.length === 0;

  return (
    <ChartShell className={className} $height={height} data-testid="candle-chart-host">
      {showSkeleton ? (
        <ChartSkeletonLayer>
          <Skeleton
            variant="chart"
            height={height}
            aria-label={t('a11y.loadingChart')}
          />
        </ChartSkeletonLayer>
      ) : null}
      <ChartContainer ref={containerRef} aria-hidden={showSkeleton} />
    </ChartShell>
  );
}
