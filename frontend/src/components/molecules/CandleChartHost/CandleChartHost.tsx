import { useEffect, useLayoutEffect, useRef } from 'react';
import {
  CandlestickSeries,
  CrosshairMode,
  LineSeries,
  createChart,
  createSeriesMarkers,
  type IChartApi,
  type ISeriesApi,
  type ISeriesMarkersPluginApi,
  type ISeriesPrimitive,
  type LogicalRange,
  type SeriesMarker,
  type Time,
} from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { CHART_LOCALIZATION, CHART_TIME_SCALE } from '@/libs/utils';
import { semanticColors } from '@/styles/tokens';
import type { CandleChartHostProps } from './CandleChartHost.types';
import {
  DEFAULT_HEIGHT,
  HISTORY_LOAD_THRESHOLD,
  INITIAL_RIGHT_PADDING,
  INITIAL_VISIBLE_BARS,
} from './CandleChartHost.constants';
import {
  ChartContainer,
  ChartShell,
  ChartSkeletonLayer,
} from './CandleChartHost.styles';
import { snapMarkersToCandleTimes } from './CandleChartHost.markers';
import {
  VertLinePrimitive,
  snapVertLinesToCandleTimes,
} from './CandleChartHost.vertLines';
import {
  candleDataSignature,
  chartPriceFormatFromCandles,
  initialVisibleLogicalRange,
  overlaysSignature,
  toCandlestickData,
  toLineData,
} from './CandleChartHost.helpers';

/**
 * Thin host for TradingView Lightweight Charts candlesticks.
 * Optional line overlays (EMA) share the price scale.
 *
 * Chart DOM stays mounted across loading and overlay toggles so the canvas
 * is not torn down. When history is prepended (more bars to the left), the
 * visible logical range is shifted so pan position stays stable.
 */
export function CandleChartHost({
  data,
  overlays = [],
  markers = [],
  vertLines = [],
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
  seriesKey = '',
  isLoadingMore = false,
  hasMoreHistory = true,
  onNeedMoreHistory,
  rightPadding = INITIAL_RIGHT_PADDING,
  anchorEndIndex,
}: CandleChartHostProps) {
  const { t } = useTranslation('common');
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());
  const vertPrimitivesRef = useRef<ISeriesPrimitive<Time>[]>([]);
  const lastVertSigRef = useRef<string>('');
  const lastCandleSigRef = useRef<string>('');
  const lastOverlaySigRef = useRef<string>('');
  const lastMarkersSigRef = useRef<string>('');
  const hasFittedRef = useRef(false);
  const suppressHistoryLoadRef = useRef(true);
  const lastPriceFormatKeyRef = useRef<string>('');
  const prevLenRef = useRef(0);
  const prevFirstTimeRef = useRef<number | null>(null);
  const seriesKeyRef = useRef(seriesKey);

  const onNeedMoreHistoryRef = useRef(onNeedMoreHistory);
  const isLoadingMoreRef = useRef(isLoadingMore);
  const hasMoreHistoryRef = useRef(hasMoreHistory);
  onNeedMoreHistoryRef.current = onNeedMoreHistory;
  isLoadingMoreRef.current = isLoadingMore;
  hasMoreHistoryRef.current = hasMoreHistory;

  // Create chart once (layout phase so the data effect below can sync same frame).
  useLayoutEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const chart = createChart(el, {
      height,
      autoSize: true,
      layout: {
        background: { color: semanticColors.chart.background },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: semanticColors.chart.grid },
        horzLines: { color: semanticColors.chart.grid },
      },
      rightPriceScale: {
        borderColor: semanticColors.border.default,
        minimumWidth: 72,
      },
      timeScale: {
        borderColor: semanticColors.border.default,
        ...CHART_TIME_SCALE,
      },
      localization: CHART_LOCALIZATION,
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: {
          color: semanticColors.border.strong,
          width: 1,
          style: 3,
          labelBackgroundColor: semanticColors.bg.elevated,
        },
        horzLine: {
          color: semanticColors.border.strong,
          width: 1,
          style: 3,
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
      priceFormat: { type: 'price', precision: 2, minMove: 0.01 },
    });

    chartRef.current = chart;
    seriesRef.current = series;
    // Markers plugin is created in a separate effect after data is available.

    const onVisibleRange = (range: LogicalRange | null) => {
      if (!range) return;
      if (suppressHistoryLoadRef.current) return;
      if (!hasMoreHistoryRef.current) return;
      if (isLoadingMoreRef.current) return;
      if (!onNeedMoreHistoryRef.current) return;
      // User panned toward older bars (left edge near start of series).
      if (range.from < HISTORY_LOAD_THRESHOLD) {
        onNeedMoreHistoryRef.current();
      }
    };
    chart.timeScale().subscribeVisibleLogicalRangeChange(onVisibleRange);

    return () => {
      chart.timeScale().unsubscribeVisibleLogicalRangeChange(onVisibleRange);
      try {
        markersPluginRef.current?.detach();
      } catch {
        // chart.remove() also tears down primitives
      }
      markersPluginRef.current = null;
      for (const p of vertPrimitivesRef.current) {
        try {
          seriesRef.current?.detachPrimitive(p);
        } catch {
          // chart.remove() also tears down primitives
        }
      }
      vertPrimitivesRef.current = [];
      lastVertSigRef.current = '';
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      overlayRefs.current.clear();
      lastCandleSigRef.current = '';
      lastOverlaySigRef.current = '';
      lastMarkersSigRef.current = '';
      hasFittedRef.current = false;
      suppressHistoryLoadRef.current = true;
      lastPriceFormatKeyRef.current = '';
      prevLenRef.current = 0;
      prevFirstTimeRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- chart lifecycle is mount/unmount only
  }, []);

  useEffect(() => {
    chartRef.current?.applyOptions({ height });
  }, [height]);

  // New symbol / interval → re-fit on next data, clear length bookkeeping.
  // Must run in useLayoutEffect *before* the data-sync layout effect so the
  // same commit does not mis-detect history-prepend or skip fitContent.
  useLayoutEffect(() => {
    if (seriesKeyRef.current === seriesKey) return;
    seriesKeyRef.current = seriesKey;
    hasFittedRef.current = false;
    suppressHistoryLoadRef.current = true;
    prevLenRef.current = 0;
    prevFirstTimeRef.current = null;
    lastCandleSigRef.current = '';
    lastOverlaySigRef.current = '';
    lastMarkersSigRef.current = '';
    lastPriceFormatKeyRef.current = '';
    lastVertSigRef.current = '';
  }, [seriesKey]);

  // Sync candles + overlays when content changes.
  useLayoutEffect(() => {
    const chart = chartRef.current;
    const candleSeries = seriesRef.current;
    if (!chart || !candleSeries) return;

    // Defensive: if seriesKey changed this commit, bookkeeping already reset above.
    if (seriesKeyRef.current !== seriesKey) {
      seriesKeyRef.current = seriesKey;
      hasFittedRef.current = false;
      suppressHistoryLoadRef.current = true;
      prevLenRef.current = 0;
      prevFirstTimeRef.current = null;
      lastCandleSigRef.current = '';
      lastOverlaySigRef.current = '';
      lastMarkersSigRef.current = '';
      lastPriceFormatKeyRef.current = '';
    }

    const candleSig = candleDataSignature(data);
    const overlaySig = overlaysSignature(overlays);
    const candlesChanged = candleSig !== lastCandleSigRef.current;
    const overlaysChanged = overlaySig !== lastOverlaySigRef.current;

    if (!candlesChanged && !overlaysChanged) {
      return;
    }

    const priceFormat = chartPriceFormatFromCandles(data);
    const priceFormatKey = `${priceFormat.precision}:${priceFormat.minMove}`;
    const priceFormatChanged = priceFormatKey !== lastPriceFormatKeyRef.current;

    if (candlesChanged || priceFormatChanged) {
      candleSeries.applyOptions({ priceFormat });
      const labelWidth = Math.min(140, Math.max(72, 48 + priceFormat.precision * 7));
      chart.applyOptions({
        rightPriceScale: { minimumWidth: labelWidth },
      });
      lastPriceFormatKeyRef.current = priceFormatKey;
    }

    if (candlesChanged) {
      const prevLen = prevLenRef.current;
      const nextLen = data.length;
      const nextFirst = data[0]?.time ?? null;
      const prevFirst = prevFirstTimeRef.current;
      const logicalRange = chart.timeScale().getVisibleLogicalRange();

      const historyPrepended =
        hasFittedRef.current &&
        prevLen > 0 &&
        nextLen > prevLen &&
        prevFirst !== null &&
        nextFirst !== null &&
        nextFirst < prevFirst;

      candleSeries.setData(toCandlestickData(data));
      lastCandleSigRef.current = candleSig;
      prevLenRef.current = nextLen;
      prevFirstTimeRef.current = nextFirst;

      // setData can invalidate series primitives — force markers re-bind next effect.
      lastMarkersSigRef.current = '';

      if (historyPrepended && logicalRange) {
        const added = nextLen - prevLen;
        chart.timeScale().setVisibleLogicalRange({
          from: logicalRange.from + added,
          to: logicalRange.to + added,
        });
      } else if (data.length > 0 && !hasFittedRef.current) {
        const range = initialVisibleLogicalRange(
          data.length,
          INITIAL_VISIBLE_BARS,
          rightPadding,
          anchorEndIndex ?? data.length,
        );
        if (range) {
          chart.timeScale().setVisibleLogicalRange(range);
        }
        hasFittedRef.current = true;
        suppressHistoryLoadRef.current = true;
        requestAnimationFrame(() => {
          suppressHistoryLoadRef.current = false;
        });
      }
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
  }, [data, overlays, seriesKey]);

  /**
   * Markers live in a separate effect so:
   * - async pump results still apply after candles (no early-return skip)
   * - setData does not leave a dead plugin (detach + recreate)
   * - zoom only changes viewport; markers stay bound to bar times
   */
  useLayoutEffect(() => {
    const candleSeries = seriesRef.current;
    if (!candleSeries) return;

    const snapped = snapMarkersToCandleTimes(markers, data)
      .slice()
      .sort((a, b) => a.time - b.time);

    const markersSig = `${candleDataSignature(data)}::${snapped
      .map((m) => `${m.time}:${m.shape}:${m.color}:${m.text ?? ''}`)
      .join('|')}`;
    if (markersSig === lastMarkersSigRef.current && markersPluginRef.current) {
      return;
    }

    // Recreate plugin after data changes — setMarkers alone can fail after setData.
    if (markersPluginRef.current) {
      try {
        markersPluginRef.current.detach();
      } catch {
        // ignore detach races during unmount
      }
      markersPluginRef.current = null;
    }

    const seriesMarkers: SeriesMarker<Time>[] = snapped.map((m) => ({
      time: m.time as Time,
      position: m.position,
      color: m.color,
      shape: m.shape,
      text: m.text,
      size: 1.5,
    }));

    markersPluginRef.current = createSeriesMarkers(candleSeries, seriesMarkers, {
      zOrder: 'top',
      autoScale: false,
    });
    lastMarkersSigRef.current = markersSig;
  }, [data, markers]);

  useLayoutEffect(() => {
    const series = seriesRef.current;
    if (!series) return;

    const snapped = snapVertLinesToCandleTimes(
      vertLines,
      data.map((c) => c.time),
    );
    const usedOnBar = new Map<number, number>();
    const stacked = snapped.map((line) => {
      const slot = usedOnBar.get(line.time) ?? 0;
      usedOnBar.set(line.time, slot + 1);
      return { ...line, labelOffsetY: slot * 20 };
    });

    const sig = `${candleDataSignature(data)}::${stacked
      .map((v) => `${v.id}:${v.time}:${v.color}:${v.label}:${v.labelOffsetY ?? 0}`)
      .join('|')}`;
    if (sig === lastVertSigRef.current) return;
    lastVertSigRef.current = sig;

    for (const p of vertPrimitivesRef.current) {
      try {
        series.detachPrimitive(p);
      } catch {
        // ignore detach races
      }
    }
    vertPrimitivesRef.current = [];

    for (const line of stacked) {
      const prim = new VertLinePrimitive(line);
      series.attachPrimitive(prim);
      vertPrimitivesRef.current.push(prim);
    }
  }, [vertLines, data]);

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
