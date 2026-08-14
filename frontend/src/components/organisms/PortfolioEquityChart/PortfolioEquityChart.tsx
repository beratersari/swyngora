import { useEffect, useLayoutEffect, useMemo, useRef } from 'react';
import { Segmented } from 'antd';
import {
  createChart,
  LineSeries,
  TickMarkType,
  type IChartApi,
  type ISeriesApi,
  type Time,
} from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { PortfolioPerformancePeriod } from '@/libs/api';
import { semanticColors } from '@/styles/tokens';
import {
  equitySpanSeconds,
  formatEquityCrosshairTime,
  formatEquityTickMark,
  toEquityLineData,
} from './PortfolioEquityChart.helpers';
import { ChartBox, ChartShell, ChartToolbar } from './PortfolioEquityChart.styles';
import type { PortfolioEquityChartProps } from './PortfolioEquityChart.types';

const PERIODS: PortfolioPerformancePeriod[] = ['1d', '1w', '1m', '3m'];

/**
 * Equity history line chart.
 * Chart remounts when period/locale/span formatting inputs change — not on every resize.
 * Height/width updates go through ResizeObserver + applyOptions so the series stays mounted.
 */
export function PortfolioEquityChart({
  points = [],
  startEquity,
  startAt,
  period,
  onPeriodChange,
  isLoading,
  isError,
  height = 220,
}: PortfolioEquityChartProps) {
  const { t, i18n } = useTranslation('portfolio');
  const boxRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const periodRef = useRef(period);
  const spanRef = useRef(0);
  const localeRef = useRef(i18n.language || 'en');
  const lineDataRef = useRef<ReturnType<typeof toEquityLineData>>([]);

  const lineData = useMemo(
    () => toEquityLineData(points, { startEquity, startAt }),
    [points, startEquity, startAt],
  );
  const spanSec = useMemo(() => equitySpanSeconds(lineData), [lineData]);
  const ready = lineData.length > 0;

  periodRef.current = period;
  spanRef.current = spanSec;
  localeRef.current = i18n.language || 'en';
  lineDataRef.current = lineData;

  const applyHostSize = (chart: IChartApi, host: HTMLElement, fallbackHeight: number) => {
    const width = host.clientWidth || host.parentElement?.clientWidth || 0;
    const hostHeight = host.clientHeight || fallbackHeight;
    // Intermediate layout frames can report 0×0 while the drawer/grid reflows;
    // applying zero size blanks the canvas until a later tick.
    if (width < 8 || hostHeight < 8) return;
    chart.applyOptions({ width, height: hostHeight });
  };

  // Create chart when the host is mounted with data; rebuild only for axis format inputs.
  // Do NOT depend on `height` — phone breakpoint height changes would remount without
  // re-running setData (lineData unchanged) and leave an empty chart until the next fetch.
  useLayoutEffect(() => {
    if (!ready) return;
    const el = boxRef.current;
    if (!el) return;

    const locale = localeRef.current;
    const chart = createChart(el, {
      width: Math.max(el.clientWidth || el.parentElement?.clientWidth || 320, 8),
      height: Math.max(el.clientHeight || height, 8),
      layout: {
        background: { color: 'transparent' },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: 'rgba(79, 212, 165, 0.08)' },
        horzLines: { color: 'rgba(79, 212, 165, 0.08)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: {
        borderVisible: false,
        timeVisible: period === '1d' || spanSec < 48 * 3600,
        secondsVisible: false,
        tickMarkFormatter: (time: Time, tickMarkType: TickMarkType) =>
          formatEquityTickMark(
            time,
            tickMarkType,
            localeRef.current,
            periodRef.current,
            spanRef.current,
          ),
      },
      localization: {
        locale,
        timeFormatter: (time: Time) => formatEquityCrosshairTime(time, localeRef.current),
      },
      handleScroll: false,
      handleScale: false,
    });
    const series = chart.addSeries(LineSeries, {
      color: semanticColors.chart.up,
      lineWidth: 2,
      pointMarkersVisible: true,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    chartRef.current = chart;
    seriesRef.current = series;

    // Seed data immediately on create (resize-only remounts used to skip the data effect).
    const initial = lineDataRef.current;
    if (initial.length > 0) {
      series.setData(initial);
      chart.timeScale().fitContent();
    }

    const ro = new ResizeObserver(() => {
      const host = boxRef.current;
      const live = chartRef.current;
      if (!host || !live) return;
      applyHostSize(live, host, height);
    });
    ro.observe(el);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
    // height intentionally omitted from deps — applied via ResizeObserver / height effect
    // eslint-disable-next-line react-hooks/exhaustive-deps -- remount only for format inputs
  }, [ready, period, i18n.language, spanSec]);

  // Responsive height from parent (phone vs desk): resize in place, never remount.
  useEffect(() => {
    const host = boxRef.current;
    const chart = chartRef.current;
    if (!host || !chart) return;
    applyHostSize(chart, host, height);
  }, [height]);

  useEffect(() => {
    const series = seriesRef.current;
    const chart = chartRef.current;
    if (!series || !chart || lineData.length === 0) return;
    series.setData(lineData);
    chart.timeScale().fitContent();
  }, [lineData]);

  return (
    <ChartShell>
      <ChartToolbar>
        <Text variant="h4" color="primary">
          {t('equityChart.title')}
        </Text>
        <Segmented
          size="small"
          value={period}
          options={PERIODS.map((p) => ({ value: p, label: t(`period.${p}`) }))}
          onChange={(v) => onPeriodChange(v as PortfolioPerformancePeriod)}
        />
      </ChartToolbar>
      {isError ? (
        <DeskEmpty title={t('equityChart.loadFailed')} />
      ) : isLoading && !ready ? (
        <Skeleton height={height} />
      ) : !ready ? (
        <DeskEmpty title={t('equityChart.empty')} />
      ) : (
        <ChartBox ref={boxRef} style={{ height }} data-testid="portfolio-equity-chart" />
      )}
    </ChartShell>
  );
}
