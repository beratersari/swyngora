import { Alert, Table, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  formatHolderAddress,
  formatHolderBalance,
  formatHolderCount,
  formatSharePct,
  holderUsdValue,
  resolveHolderBalance,
} from './helpers';
import { AddressCell, AddressLabel, AddressWrap, Panel, StatCard, StatsGrid, TitleRow } from './HolderPanel.styles';
import type { HolderPanelProps } from './HolderPanel.types';

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

export function HolderPanel({
  holders,
  error,
  isLoading = false,
  circulatingSupply,
  priceUsd,
}: HolderPanelProps) {
  const { t, i18n } = useTranslation('detail');
  const { formatCompact } = useDisplayCurrency();
  const rows = holders?.topHolders ?? [];
  const showUsd = rows.some(
    (row) => holderUsdValue(row.sharePct, circulatingSupply, priceUsd) != null,
  );
  const showStats = Boolean(holders) && !error;

  return (
    <Panel>
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('holders.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('holders.subtitle')}
        </Text>
      </TitleRow>

      {error ? (
        <Alert type="info" showIcon message={t('holders.title')} description={error} />
      ) : null}
      {!error && holders?.stale ? (
        <Alert type="warning" showIcon message={t('holders.stale')} />
      ) : null}

      {showStats ? (
      <StatsGrid>
        <Stat
          label={t('holders.count')}
          value={formatHolderCount(holders?.holderCount)}
          isLoading={isLoading}
        />
        <Stat
          label={t('holders.dailyActive')}
          value={formatHolderCount(holders?.dailyActive)}
          isLoading={isLoading}
        />
        <Stat
          label={t('holders.topTen')}
          value={formatSharePct(holders?.topTenSharePct)}
          isLoading={isLoading}
        />
        <Stat
          label={t('holders.topFifty')}
          value={formatSharePct(holders?.topFiftySharePct)}
          isLoading={isLoading}
        />
        <Stat
          label={t('holders.topHundred')}
          value={formatSharePct(holders?.topHundredSharePct)}
          isLoading={isLoading}
        />
      </StatsGrid>
      ) : null}

      {showStats && rows.length === 0 && !isLoading ? (
        <Text variant="caption" color="secondary">
          {t('holders.emptyWallets')}
        </Text>
      ) : null}

      {rows.length > 0 ? (
        <Table
          size="small"
          pagination={false}
          rowKey={(row) => row.address ?? ''}
          dataSource={rows}
          columns={[
            {
              title: t('holders.address'),
              dataIndex: 'address',
              render: (addr: string, row: { label?: string }) => (
                <AddressWrap>
                  {row.label ? <AddressLabel>{row.label}</AddressLabel> : null}
                  <Typography.Text copyable={{ text: addr }}>
                    <AddressCell>{formatHolderAddress(addr)}</AddressCell>
                  </Typography.Text>
                </AddressWrap>
              ),
            },
            {
              title: t('holders.balance'),
              dataIndex: 'balance',
              align: 'right',
              render: (v: number, row: { sharePct?: number }) =>
                formatHolderBalance(resolveHolderBalance(v, row.sharePct, circulatingSupply)),
            },
            ...(showUsd
              ? [
                  {
                    title: t('holders.valueUsd'),
                    key: 'usd',
                    align: 'right' as const,
                    render: (_: unknown, row: { sharePct?: number }) =>
                      formatCompact(
                        holderUsdValue(row.sharePct, circulatingSupply, priceUsd),
                        'USD',
                      ),
                  },
                ]
              : []),
            {
              title: t('holders.share'),
              dataIndex: 'sharePct',
              align: 'right',
              render: (v: number) => formatSharePct(v),
            },
          ]}
        />
      ) : null}

      {holders?.asOf || holders?.source ? (
        <Text variant="caption" color="secondary">
          {t('holders.source', { source: holders.source ?? 'coinmarketcap' })}
          {holders.asOf
            ? ` · ${t('stats.asOf', {
                date: new Date(holders.asOf).toLocaleString(i18n.language, {
                  dateStyle: 'medium',
                  timeStyle: 'short',
                }),
              })}`
            : null}
        </Text>
      ) : null}
    </Panel>
  );
}
