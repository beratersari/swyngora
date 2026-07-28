import { useEffect, useRef } from 'react';
import { Platform, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { LineSeries, createChart, type IChartApi, type UTCTimestamp } from 'lightweight-charts';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { semanticColors } from '@/styles/tokens';
import type { IndicatorRsiPaneProps } from './IndicatorRsiPane.types';
import { styles } from './IndicatorRsiPane.styles';

export function IndicatorRsiPane({
  data,
  latestRsi,
  isLoading,
  errorMessage,
  height = 140,
}: IndicatorRsiPaneProps) {
  const { t } = useTranslation('detail');
  const hostRef = useRef<HTMLDivElement | null>(null);
  const rsiTitle = t('rsiTitle');

  useEffect(() => {
    if (Platform.OS !== 'web' || !hostRef.current || !data.length || isLoading) return;
    const el = hostRef.current;
    const chart: IChartApi = createChart(el, {
      height,
      width: el.clientWidth || 320,
      layout: {
        background: { color: semanticColors.bg.muted },
        textColor: semanticColors.text.secondary,
      },
      rightPriceScale: { borderVisible: false, scaleMargins: { top: 0.1, bottom: 0.1 } },
      timeScale: { borderVisible: false },
      grid: {
        vertLines: { color: 'rgba(112, 125, 125, 0.15)' },
        horzLines: { color: 'rgba(112, 125, 125, 0.15)' },
      },
    });
    const series = chart.addSeries(LineSeries, {
      color: '#74F9BC',
      lineWidth: 2,
      title: rsiTitle,
    });
    series.setData(
      data.map((p) => ({
        time: p.time as UTCTimestamp,
        value: p.value,
      })),
    );
    series.createPriceLine({
      price: 70,
      color: 'rgba(224, 122, 122, 0.6)',
      lineWidth: 1,
      lineStyle: 2,
      axisLabelVisible: true,
      title: '70',
    });
    series.createPriceLine({
      price: 30,
      color: 'rgba(111, 191, 138, 0.6)',
      lineWidth: 1,
      lineStyle: 2,
      axisLabelVisible: true,
      title: '30',
    });
    chart.timeScale().fitContent();
    return () => chart.remove();
  }, [data, height, isLoading, rsiTitle]);

  const latestLabel =
    latestRsi !== null && Number.isFinite(latestRsi) ? latestRsi.toFixed(2) : '—';

  return (
    <View style={styles.card}>
      <Text variant="label" color="secondary">
        {t('rsiLatest', { value: latestLabel })}
      </Text>
      {isLoading && !data.length ? (
        <View style={styles.center}>
          <Skeleton height={height - 20} width="100%" />
        </View>
      ) : errorMessage ? (
        <Text variant="caption" color="error">
          {errorMessage}
        </Text>
      ) : !data.length ? (
        <Text variant="caption" color="secondary">
          {t('rsiEmpty')}
        </Text>
      ) : Platform.OS === 'web' ? (
        <div ref={hostRef} style={{ width: '100%', height }} />
      ) : (
        <Text variant="caption" color="secondary">
          {t('rsiWebOnly')}
        </Text>
      )}
      <Text variant="caption" color="steel">
        {t('rsiBandsHint')}
      </Text>
    </View>
  );
}
