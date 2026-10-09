import { useEffect, useLayoutEffect, useRef } from 'react';
import {
  CandlestickSeries,
  CrosshairMode,
  HistogramSeries,
  LineSeries,
  LineStyle,
  createChart,
  createSeriesMarkers,
  type IChartApi,
  type IPriceLine,
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
import type { CandleChartHostProps, CandleChartLowerSeries } from './CandleChartHost.types';
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
import { DeskOverlayPrimitive } from './CandleChartHost.desk';
import {
  candleDataSignature,
  chartPriceFormatFromCandles,
  initialVisibleLogicalRange,
  logicalRangeCovering,
  overlaysSignature,
  toCandlestickData,
  toHistogramData,
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
  lowerSeries = [],
  priceLines = [],
  bookHeat = [],
  footprint = [],
  candleFootprints = [],
  footprintStepSec = 0,
  liqProfile = [],
  liqHeat = [],
  openLevels = [],
  onVisibleSpan,
  markers = [],
  vertLines = [],
  barDurationSec,
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
  seriesKey = '',
  isLoadingMore = false,
  hasMoreHistory = true,
  onNeedMoreHistory,
  rightPadding = INITIAL_RIGHT_PADDING,
  anchorEndIndex,
  onBarTimeClick,
}: CandleChartHostProps) {
  const { t } = useTranslation('common');
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());
  const lowerRefs = useRef<Map<string, ISeriesApi<'Line'> | ISeriesApi<'Histogram'>>>(new Map());
  const lowerPaneRef = useRef<Map<string, number>>(new Map());
  const lowerKindRef = useRef<Map<string, CandleChartLowerSeries['kind']>>(new Map());
  const lastLowerSigRef = useRef('');
  const priceLineRefs = useRef<IPriceLine[]>([]);
  const lastPriceLineSigRef = useRef('');
  const vertPrimitivesRef = useRef<ISeriesPrimitive<Time>[]>([]);
  const deskRef = useRef<DeskOverlayPrimitive | null>(null);
  const lastVertSigRef = useRef<string>('');
  const lastCandleSigRef = useRef<string>('');
  const lastOverlaySigRef = useRef<string>('');
  const lastMarkersSigRef = useRef<string>('');
  const hasFittedRef = useRef(false);
  /** True after a drag or wheel. Later panes must not pull the view back. */
  const userPannedRef = useRef(false);
  const pointerDownRef = useRef(false);
  const anchorEndRef = useRef(anchorEndIndex);
  const rightPaddingRef = useRef(rightPadding);
  anchorEndRef.current = anchorEndIndex;
  rightPaddingRef.current = rightPadding;
  const suppressHistoryLoadRef = useRef(true);
  const pinLatestRef = useRef<(attempt?: number) => void>(() => {});
  pinLatestRef.current = (attempt = 0) => {
    const unlockHistory = () => {
      if (!suppressHistoryLoadRef.current) return;
      requestAnimationFrame(() => {
        suppressHistoryLoadRef.current = false;
      });
    };
    const chart = chartRef.current;
    if (!chart || userPannedRef.current) {
      unlockHistory();
      return;
    }
    const bars = dataRef.current;
    if (bars.length === 0) return;
    const anchorEnd = anchorEndRef.current ?? bars.length;
    const anchorCount = Math.min(bars.length, Math.max(0, anchorEnd));
    const anchorBar = bars[(anchorCount || bars.length) - 1];
    const endBar = bars[bars.length - 1];
    if (!anchorBar || !endBar) return;
    const scale = chart.timeScale();
    const pad = rightPaddingRef.current;
    const applyRange = (range: { from: number; to: number }) => {
      suppressHistoryLoadRef.current = true;
      scale.setVisibleLogicalRange(range);
      hasFittedRef.current = true;
      requestAnimationFrame(() => {
        suppressHistoryLoadRef.current = false;
      });
    };
    // Tests and older stubs have no time-scale index. Fall back to candle order.
    if (typeof scale.timeToIndex !== 'function') {
      if (hasFittedRef.current) {
        unlockHistory();
        return;
      }
      const range = initialVisibleLogicalRange(
        bars.length,
        INITIAL_VISIBLE_BARS,
        pad,
        anchorEnd,
      );
      if (range) applyRange(range);
      else unlockHistory();
      return;
    }
    const anchorIndex = scale.timeToIndex(anchorBar.time as Time, false);
    const endIndex = scale.timeToIndex(endBar.time as Time, false);
    if (anchorIndex == null || endIndex == null) {
      if (attempt < 3) {
        requestAnimationFrame(() => pinLatestRef.current(attempt + 1));
        return;
      }
      unlockHistory();
      return;
    }
    const range = logicalRangeCovering(anchorIndex, endIndex, INITIAL_VISIBLE_BARS, pad);
    if (!range) {
      unlockHistory();
      return;
    }
    const current = scale.getVisibleLogicalRange();
    if (
      current &&
      Math.abs(current.from - range.from) < 0.25 &&
      Math.abs(current.to - range.to) < 0.25
    ) {
      hasFittedRef.current = true;
      unlockHistory();
      return;
    }
    applyRange(range);
  };
  const lastPriceFormatKeyRef = useRef<string>('');
  const prevLenRef = useRef(0);
  const prevFirstTimeRef = useRef<number | null>(null);
  const seriesKeyRef = useRef(seriesKey);

  const onNeedMoreHistoryRef = useRef(onNeedMoreHistory);
  const isLoadingMoreRef = useRef(isLoadingMore);
  const hasMoreHistoryRef = useRef(hasMoreHistory);
  const onVisibleSpanRef = useRef(onVisibleSpan);
  const onBarTimeClickRef = useRef(onBarTimeClick);
  const dataRef = useRef(data);
  onNeedMoreHistoryRef.current = onNeedMoreHistory;
  isLoadingMoreRef.current = isLoadingMore;
  hasMoreHistoryRef.current = hasMoreHistory;
  onVisibleSpanRef.current = onVisibleSpan;
  onBarTimeClickRef.current = onBarTimeClick;
  dataRef.current = data;

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
    const desk = new DeskOverlayPrimitive();
    series.attachPrimitive(desk);
    deskRef.current = desk;
    // Markers plugin is created in a separate effect after data is available.

    const onPointerDown = () => {
      pointerDownRef.current = true;
    };
    const onPointerUp = () => {
      pointerDownRef.current = false;
    };
    const onPointerMove = (event: PointerEvent) => {
      if (pointerDownRef.current && event.buttons !== 0) userPannedRef.current = true;
    };
    const onWheel = () => {
      userPannedRef.current = true;
    };
    el.addEventListener('pointerdown', onPointerDown);
    el.addEventListener('pointerup', onPointerUp);
    el.addEventListener('pointercancel', onPointerUp);
    el.addEventListener('pointermove', onPointerMove);
    el.addEventListener('wheel', onWheel, { passive: true });

    const onVisibleRange = (range: LogicalRange | null) => {
      if (!range) return;
      const visible = chart.timeScale().getVisibleRange?.();
      if (
        visible &&
        typeof visible.from === 'number' &&
        typeof visible.to === 'number' &&
        visible.to > visible.from &&
        onVisibleSpanRef.current
      ) {
        onVisibleSpanRef.current(visible.from, visible.to);
      }
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

    const onChartClick = (param: { time?: Time; point?: unknown }) => {
      if (param.point == null || typeof param.time !== 'number' || !Number.isFinite(param.time)) return;
      onBarTimeClickRef.current?.(param.time);
    };
    if (typeof chart.subscribeClick === 'function') {
      chart.subscribeClick(onChartClick);
    }

    return () => {
      if (typeof chart.unsubscribeClick === 'function') {
        chart.unsubscribeClick(onChartClick);
      }
      el.removeEventListener('pointerdown', onPointerDown);
      el.removeEventListener('pointerup', onPointerUp);
      el.removeEventListener('pointercancel', onPointerUp);
      el.removeEventListener('pointermove', onPointerMove);
      el.removeEventListener('wheel', onWheel);
      chart.timeScale().unsubscribeVisibleLogicalRangeChange(onVisibleRange);
      try {
        markersPluginRef.current?.detach();
      } catch {
        // chart.remove() also tears down primitives
      }
      markersPluginRef.current = null;
      deskRef.current = null;
      try {
        series.detachPrimitive(desk);
      } catch {
        /* chart already removed */
      }
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
      lowerRefs.current.clear();
      lowerPaneRef.current.clear();
      lowerKindRef.current.clear();
      lastLowerSigRef.current = '';
      priceLineRefs.current = [];
      lastPriceLineSigRef.current = '';
      lastCandleSigRef.current = '';
      lastOverlaySigRef.current = '';
      lastMarkersSigRef.current = '';
      hasFittedRef.current = false;
      userPannedRef.current = false;
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

  useEffect(() => {
    deskRef.current?.setModel({
      bookHeat,
      footprint,
      candleFootprints,
      footprintStepSec,
      liqProfile,
      liqHeat,
      openLevels,
    });
  }, [bookHeat, footprint, candleFootprints, footprintStepSec, liqProfile, liqHeat, openLevels]);

  // New symbol / interval → re-fit on next data, clear length bookkeeping.
  // Must run in useLayoutEffect *before* the data-sync layout effect so the
  // same commit does not mis-detect history-prepend or skip fitContent.
  useLayoutEffect(() => {
    if (seriesKeyRef.current === seriesKey) return;
    seriesKeyRef.current = seriesKey;
    hasFittedRef.current = false;
    userPannedRef.current = false;
    suppressHistoryLoadRef.current = true;
    prevLenRef.current = 0;
    prevFirstTimeRef.current = null;
    lastCandleSigRef.current = '';
    lastOverlaySigRef.current = '';
    lastMarkersSigRef.current = '';
    lastPriceFormatKeyRef.current = '';
    lastVertSigRef.current = '';
    // Force the pane and price-line effects to rebind after a new symbol.
    lastLowerSigRef.current = 'pending';
    lastPriceLineSigRef.current = 'pending';
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
      userPannedRef.current = false;
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

      // A pan that the user started keeps its place when older candles arrive.
      // Otherwise the pin below puts the latest candle back on the right.
      if (historyPrepended && logicalRange && userPannedRef.current) {
        const added = nextLen - prevLen;
        chart.timeScale().setVisibleLogicalRange({
          from: logicalRange.from + added,
          to: logicalRange.to + added,
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

    // EMA and other overlays insert times the candle count does not know about.
    // Pin after that data is on the scale, or the latest candles sit off to the right.
    if (data.length > 0) pinLatestRef.current();
  }, [data, overlays, seriesKey]);

  useLayoutEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    const sig = lowerSeries
      .map((s) => {
        let sum = 0;
        for (const point of s.data) sum += point.value;
        const last = s.data[s.data.length - 1];
        return `${s.id}:${s.pane}:${s.kind}:${s.color}:${s.downColor ?? ''}:${s.title ?? ''}:${s.data.length}:${s.data[0]?.time ?? ''}:${last?.time ?? ''}:${sum}`;
      })
      .join('||');
    if (sig === lastLowerSigRef.current) return;
    lastLowerSigRef.current = sig;

    const nextIds = new Set(lowerSeries.map((s) => s.id));
    for (const [id, series] of lowerRefs.current) {
      const next = lowerSeries.find((s) => s.id === id);
      const paneChanged = next != null && lowerPaneRef.current.get(id) !== next.pane;
      const kindChanged = next != null && lowerKindRef.current.get(id) !== next.kind;
      if (!nextIds.has(id) || paneChanged || kindChanged) {
        chart.removeSeries(series);
        lowerRefs.current.delete(id);
        lowerPaneRef.current.delete(id);
        lowerKindRef.current.delete(id);
      }
    }

    for (const s of lowerSeries) {
      let series = lowerRefs.current.get(s.id);
      if (!series) {
        const common = {
          color: s.color,
          title: s.title ?? s.id,
          priceLineVisible: false,
          lastValueVisible: true,
          priceFormat: { type: 'volume' as const },
        };
        series =
          s.kind === 'histogram'
            ? chart.addSeries(HistogramSeries, common, s.pane)
            : chart.addSeries(LineSeries, { ...common, lineWidth: 2 }, s.pane);
        lowerRefs.current.set(s.id, series);
        lowerPaneRef.current.set(s.id, s.pane);
        lowerKindRef.current.set(s.id, s.kind);
      } else {
        series.applyOptions({ color: s.color, title: s.title ?? s.id });
      }
      if (s.kind === 'histogram') {
        const down = s.downColor ?? s.color;
        (series as ISeriesApi<'Histogram'>).setData(
          toHistogramData(
            s.data.map((point) => ({
              ...point,
              color: point.value < 0 ? down : s.color,
            })),
          ),
        );
      } else {
        (series as ISeriesApi<'Line'>).setData(toLineData(s.data));
      }
    }

    const panes = chart.panes();
    panes[0]?.setStretchFactor(3);
    for (let i = 1; i < panes.length; i += 1) {
      panes[i]?.setStretchFactor(1);
    }
    // Lower panes add their own timestamps. Re-pin so those points stay in view
    // until the user scrolls.
    pinLatestRef.current();
  }, [lowerSeries, seriesKey]);

  useLayoutEffect(() => {
    const series = seriesRef.current;
    if (!series) return;

    const sig = priceLines
      .map((line) => `${line.id}:${line.price}:${line.color}:${line.title}:${line.lineWidth ?? 1}`)
      .join('|');
    if (sig === lastPriceLineSigRef.current) return;
    lastPriceLineSigRef.current = sig;

    for (const line of priceLineRefs.current) {
      try {
        series.removePriceLine(line);
      } catch {
        // series may already have dropped the line
      }
    }
    priceLineRefs.current = [];

    const prices: number[] = [];
    for (const line of priceLines) {
      if (!Number.isFinite(line.price)) continue;
      prices.push(line.price);
      priceLineRefs.current.push(
        series.createPriceLine({
          price: line.price,
          color: line.color,
          lineWidth: line.lineWidth ?? 1,
          lineStyle: LineStyle.Dashed,
          axisLabelVisible: true,
          title: line.title,
        }),
      );
    }
    // Keep 10×–25× pockets on screen when they sit near the candles.
    // Farther than 12% stays off the scale so one distant level cannot flatten price.
    if (prices.length === 0) {
      series.applyOptions({ autoscaleInfoProvider: undefined });
      return;
    }
    series.applyOptions({
      autoscaleInfoProvider: (original) => {
        const base = original();
        if (!base?.priceRange) return base;
        let min = base.priceRange.minValue;
        let max = base.priceRange.maxValue;
        const mid = (min + max) / 2;
        const cap = Math.abs(mid) * 0.12;
        for (const price of prices) {
          if (Math.abs(price - mid) > cap) continue;
          min = Math.min(min, price);
          max = Math.max(max, price);
        }
        return { ...base, priceRange: { minValue: min, maxValue: max } };
      },
    });
  }, [priceLines, seriesKey]);

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
      { barDurationSec },
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
  }, [vertLines, data, barDurationSec]);

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
