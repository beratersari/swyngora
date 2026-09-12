import { Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatArbMoney, formatArbPct, routeLabel } from './helpers';
import type { PriceDiffScanTableProps } from './PriceDiffScanTable.types';

export function PriceDiffScanTable({ scan, isLoading }: PriceDiffScanTableProps) {
  const { t } = useTranslation('arb');
  const rows = scan?.routes ?? [];
  return (
    <Table
      size="small"
      rowKey={(r) => `${r.buyExchange}-${r.sellExchange}-${r.profitAfterFees ?? ''}`}
      loading={isLoading}
      pagination={false}
      dataSource={rows}
      locale={{ emptyText: t('price.empty') }}
      columns={[
        {
          title: t('price.route'),
          render: (_, r) => <Text variant="bodySm">{routeLabel(r)}</Text>,
        },
        {
          title: t('price.profit'),
          render: (_, r) => (
            <Text variant="numeric">{formatArbMoney(r.profitAfterFees)}</Text>
          ),
        },
        {
          title: t('price.profitPct'),
          render: (_, r) => <Text variant="numeric">{formatArbPct(r.profitPct)}</Text>,
        },
        {
          title: t('price.used'),
          render: (_, r) => <Text variant="numeric">{formatArbMoney(r.usedNotional)}</Text>,
        },
        {
          title: t('price.executable'),
          render: (_, r) => (
            <Text variant="caption">{r.executable ? t('price.yes') : t('price.no')}</Text>
          ),
        },
      ]}
    />
  );
}
