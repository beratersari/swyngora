import { useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Segmented } from 'antd';
import { createChart, LineSeries, type IChartApi, type ISeriesApi, type Time } from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { semanticColors } from '@/styles/tokens';
import { BASIS_CURVE_RANGES, BASIS_CURVE_WINDOWS } from './constants';
import { curveWindow, hasCurveHistory, moveKindLabel, toCurveLine, windowChangeLabel } from './helpers';
import { ChartBox, Chip, Chips, Moves, Stack, Toolbar } from './BasisCurveHistory.styles';
import type { BasisCurveHistoryProps, BasisCurveHistoryRange } from './BasisCurveHistory.types';

export function BasisCurveHistory({ curve, venueLabel }: BasisCurveHistoryProps) {
  const { t } = useTranslation(['basis', 'common']);
  const [range, setRange] = useState<BasisCurveHistoryRange>('7d');
  const boxRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const lineRef = useRef<ReturnType<typeof toCurveLine>>([]);

  const history = curve?.history;
  const line = useMemo(() => toCurveLine(history, range), [history, range]);
  lineRef.current = line;
  const ready = hasCurveHistory(history, range) && line.length > 0;
  const moves = curve?.moves ?? [];

  useLayoutEffect(() => {
    if (!ready) return;
    const el = boxRef.current;
    if (!el) return;
    const chart = createChart(el, {
      width: Math.max(el.clientWidth || 320, 8),
      height: Math.max(el.clientHeight || 200, 8),
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
        timeVisible: range !== '7d',
        secondsVisible: false,
      },
      handleScroll: false,
      handleScale: false,
    });
    const series = chart.addSeries(LineSeries, {
      color: semanticColors.chart.emaFast,
      lineWidth: 2,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    chartRef.current = chart;
    seriesRef.current = series;
    const initial = lineRef.current;
    if (initial.length > 0) {
      series.setData(initial.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    const ro = new ResizeObserver(() => {
      if (!boxRef.current || !chartRef.current) return;
      chartRef.current.applyOptions({
        width: Math.max(boxRef.current.clientWidth || 320, 8),
        height: Math.max(boxRef.current.clientHeight || 200, 8),
      });
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [ready, range]);

  useLayoutEffect(() => {
    const series = seriesRef.current;
    if (!series || !ready) return;
    series.setData(line.map((p) => ({ time: p.time as Time, value: p.value })));
  }, [line, ready]);

  if (!ready && moves.length === 0 && !(curve?.windows ?? []).some((w) => w.complete)) {
    return null;
  }

  return (
    <Stack data-testid="basis-curve-history">
      <Toolbar>
        <Text variant="bodySm" weight={700}>
          {venueLabel ? `${venueLabel} · ` : ''}
          {t('basis:overTime')}
        </Text>
        {ready ? (
          <Segmented
            size="small"
            value={range}
            options={BASIS_CURVE_RANGES.map((id) => ({ label: t(`basis:ranges.${id}`), value: id }))}
            onChange={(next) => setRange(next as BasisCurveHistoryRange)}
          />
        ) : null}
      </Toolbar>
      <Text variant="caption" color="secondary">
        {t('basis:overTimeHint')}
      </Text>
      <Chips>
        {BASIS_CURVE_WINDOWS.map((id) => {
          const win = curveWindow(curve?.windows, id);
          const label = windowChangeLabel(win);
          return (
            <Chip key={id}>
              <Text variant="caption" color="secondary">
                {id}
              </Text>
              <Text variant="bodySm" weight={700}>
                {win?.complete ? t(`basis:curveChanges.${win.change || 'stable'}`) : '—'}
              </Text>
              {label ? (
                <Text variant="caption" color="secondary">
                  {label}
                </Text>
              ) : null}
            </Chip>
          );
        })}
      </Chips>
      {ready ? <ChartBox ref={boxRef} data-testid="basis-curve-history-chart" /> : null}
      {moves.length > 0 ? (
        <Moves>
          {moves.map((m, i) => (
            <li key={`${m.time}-${m.kind}-${i}`}>
              <Text variant="caption" weight={700}>
                {t(`basis:curveMoves.${moveKindLabel(m.kind)}`)}
              </Text>{' '}
              <Text variant="caption" color="secondary">
                {m.summary}
              </Text>
            </li>
          ))}
        </Moves>
      ) : null}
    </Stack>
  );
}
