import { Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatArbMoney, formatArbPct } from '@/components/organisms/PriceDiffScanTable';
import type { FundingArbScanTableProps } from './FundingArbScanTable.types';

export function FundingArbScanTable({ scan, isLoading }: FundingArbScanTableProps) {
  const { t } = useTranslation('arb');
  const rows = scan?.hits ?? [];
  return (
    <Table
      size="small"
      rowKey={(r) => `${r.symbol}-${r.longExchange}-${r.shortExchange}`}
      loading={isLoading}
      pagination={false}
      dataSource={rows}
      locale={{ emptyText: t('funding.empty') }}
      columns={[
        {
          title: t('funding.symbol'),
          render: (_, r) => <Text variant="bodySm">{r.symbol ?? '—'}</Text>,
        },
        {
          title: t('funding.long'),
          render: (_, r) => <Text variant="bodySm">{r.longExchange ?? '—'}</Text>,
        },
        {
          title: t('funding.short'),
          render: (_, r) => <Text variant="bodySm">{r.shortExchange ?? '—'}</Text>,
        },
        {
          title: t('funding.spread'),
          render: (_, r) => <Text variant="numeric">{formatArbPct(r.spreadPct)}</Text>,
        },
        {
          title: t('funding.net'),
          render: (_, r) => (
            <Text variant="numeric">{formatArbMoney(r.netHorizonAfterRoundTrip)}</Text>
          ),
        },
        {
          title: t('funding.read'),
          render: (_, r) => <Text variant="caption">{r.summary ?? '—'}</Text>,
        },
      ]}
    />
  );
}
