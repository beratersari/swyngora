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
 * Chart remounts when period/data readiness changes so axis formatters stay correct.
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

  const lineData = useMemo(
    () => toEquityLineData(points, { startEquity, startAt }),
    [points, startEquity, startAt],
  );
  const spanSec = useMemo(() => equitySpanSeconds(lineData), [lineData]);
  const ready = lineData.length > 0;

  periodRef.current = period;
  spanRef.current = spanSec;
  localeRef.current = i18n.language || 'en';

  // Create chart only when the host is in the DOM; rebuild on period/locale so axis format updates.
  useLayoutEffect(() => {
    if (!ready) return;
    const el = boxRef.current;
    if (!el) return;

    const locale = localeRef.current;
    const chart = createChart(el, {
      width: el.clientWidth || el.parentElement?.clientWidth || 320,
      height,
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

    const ro = new ResizeObserver(() => {
      const host = boxRef.current;
      if (!host || !chartRef.current) return;
      chartRef.current.applyOptions({ width: host.clientWidth, height });
    });
    ro.observe(el);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [ready, height, period, i18n.language, spanSec]);

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
