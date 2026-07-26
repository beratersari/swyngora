import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  formatCompactUsd,
  formatPrice,
  formatTradeCount,
} from '@/libs/utils';
import { formatMaxSupply } from './DetailStats.helpers';
import { StatCard, StatsGrid, StatsSection } from './DetailStats.styles';
import type { DetailStatsProps } from './DetailStats.types';

function Stat({
  label,
  value,
  isLoading,
}: {
  label: string;
  value: string;
  isLoading?: boolean;
}) {
  return (
    <StatCard>
      <Text variant="caption" color="secondary">
        {label}
      </Text>
      <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
        {value}
      </Text>
    </StatCard>
  );
}

function formatSupplyNum(v: number | null | undefined): string {
  if (v === null || v === undefined || !Number.isFinite(v)) return '—';
  const abs = Math.abs(v);
  const sign = v < 0 ? '-' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  return v.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

export function DetailStats({
  exchange,
  ticker,
  supply,
  tickerError,
  supplyError,
  isLoading = false,
}: DetailStatsProps) {
  const { t, i18n } = useTranslation('detail');

  return (
    <StatsSection>
      {tickerError ? (
        <Alert type="warning" showIcon message={t('stats.tickerTitle')} description={tickerError} />
      ) : null}
      {supplyError ? (
        <Alert type="info" showIcon message={t('stats.supplyTitle')} description={supplyError} />
      ) : null}

      <StatsGrid>
        <Stat label={t('stats.open')} value={formatPrice(ticker?.openPrice)} isLoading={isLoading} />
        <Stat label={t('stats.high24h')} value={formatPrice(ticker?.highPrice)} isLoading={isLoading} />
        <Stat label={t('stats.low24h')} value={formatPrice(ticker?.lowPrice)} isLoading={isLoading} />
        <Stat
          label={t('stats.baseVol')}
          value={formatCompactUsd(ticker?.volume)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.quoteVol')}
          value={formatCompactUsd(ticker?.quoteVolume)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.trades24h')}
          value={formatTradeCount(ticker?.tradeCount, exchange)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.circSupply')}
          value={formatSupplyNum(supply?.circulatingSupply)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.totalSupply')}
          value={formatSupplyNum(supply?.totalSupply)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.maxSupply')}
          value={formatMaxSupply(supply, t('stats.maxSupplyOpen'), formatSupplyNum)}
          isLoading={isLoading}
        />
        <Stat
          label={t('stats.usdSnap')}
          value={formatPrice(supply?.currentPriceUsd)}
          isLoading={isLoading}
        />
      </StatsGrid>
      {supply?.asOf || supply?.source ? (
        <Text variant="caption" color="secondary">
          {t('stats.supplySource', { source: supply.source ?? 'binance' })}
          {supply.asOf
            ? ` · ${t('stats.asOf', {
                date: new Date(supply.asOf).toLocaleString(i18n.language, {
                  dateStyle: 'medium',
                  timeStyle: 'short',
                }),
              })}`
            : null}
        </Text>
      ) : null}
    </StatsSection>
  );
}
