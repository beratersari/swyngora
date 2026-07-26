import { Alert } from 'antd';
import { Text } from '@/components/atoms/Text';
import {
  formatCompactUsd,
  formatPrice,
  formatTradeCount,
} from '@/libs/utils';
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
      <Text variant="caption" color="steel">
        {label}
      </Text>
      <Text variant="numeric" color="cream" isLoading={isLoading} skeletonWidth="70%">
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
  return (
    <StatsSection>
      {tickerError ? (
        <Alert type="warning" showIcon message="24h ticker" description={tickerError} />
      ) : null}
      {supplyError ? (
        <Alert type="info" showIcon message="Supply" description={supplyError} />
      ) : null}

      <StatsGrid>
        <Stat label="Open" value={formatPrice(ticker?.openPrice)} isLoading={isLoading} />
        <Stat label="High 24h" value={formatPrice(ticker?.highPrice)} isLoading={isLoading} />
        <Stat label="Low 24h" value={formatPrice(ticker?.lowPrice)} isLoading={isLoading} />
        <Stat
          label="Base vol"
          value={formatCompactUsd(ticker?.volume)}
          isLoading={isLoading}
        />
        <Stat
          label="Quote vol"
          value={formatCompactUsd(ticker?.quoteVolume)}
          isLoading={isLoading}
        />
        <Stat
          label="Trades 24h"
          value={formatTradeCount(ticker?.tradeCount, exchange)}
          isLoading={isLoading}
        />
        <Stat
          label="Circ. supply"
          value={formatSupplyNum(supply?.circulatingSupply)}
          isLoading={isLoading}
        />
        <Stat
          label="Total supply"
          value={formatSupplyNum(supply?.totalSupply)}
          isLoading={isLoading}
        />
        <Stat
          label="Max supply"
          value={
            supply?.maxSupply === null || supply?.maxSupply === undefined
              ? '∞ / n/a'
              : formatSupplyNum(supply.maxSupply)
          }
          isLoading={isLoading}
        />
        <Stat
          label="USD (supply snap)"
          value={formatPrice(supply?.currentPriceUsd)}
          isLoading={isLoading}
        />
      </StatsGrid>
      {supply?.asOf || supply?.source ? (
        <Text variant="caption" color="secondary">
          Supply source: {supply.source ?? 'binance'}
          {supply.asOf
            ? ` · as of ${new Date(supply.asOf).toLocaleString(undefined, {
                dateStyle: 'medium',
                timeStyle: 'short',
              })}`
            : null}
        </Text>
      ) : null}
    </StatsSection>
  );
}
