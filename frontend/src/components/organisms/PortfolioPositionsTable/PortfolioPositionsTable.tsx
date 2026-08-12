import { Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { SpotPosition } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioPositionsTableProps } from './PortfolioPositionsTable.types';

export function PortfolioPositionsTable({ items, loading, onOpen }: PortfolioPositionsTableProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const columns: ColumnsType<SpotPosition> = [
    {
      title: t('portfolio:positions.symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Text variant="label" mono color="primary">
          {formatSymbolDisplay(row.symbol)}
        </Text>
      ),
    },
    {
      title: t('portfolio:positions.exchange'),
      dataIndex: 'exchange',
      key: 'exchange',
      render: (ex: string | undefined) =>
        ex ? <BrandTag variant="exchange">{ex}</BrandTag> : '—',
    },
    {
      title: t('portfolio:positions.qty'),
      dataIndex: 'quantity',
      key: 'qty',
      align: 'right',
      render: (v: number | undefined) => (
        <Text variant="body" mono>
          {formatPrice(v)}
        </Text>
      ),
    },
    {
      title: t('portfolio:positions.avgCost'),
      dataIndex: 'avgCost',
      key: 'avg',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:positions.mark'),
      dataIndex: 'markPrice',
      key: 'mark',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:positions.value'),
      dataIndex: 'marketValue',
      key: 'val',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:positions.uPnl'),
      dataIndex: 'unrealizedPnL',
      key: 'upnl',
      align: 'right',
      render: (v: number | undefined) => (
        <Text variant="body" mono color={v != null && v < 0 ? 'error' : v != null && v > 0 ? 'success' : 'primary'}>
          {formatPrice(v)}
        </Text>
      ),
    },
    {
      title: t('portfolio:positions.open'),
      key: 'open',
      render: (_, row) =>
        onOpen && row.exchange && row.symbol ? (
          <Button size="small" type="link" onClick={() => onOpen(row.exchange!, row.symbol!)}>
            {t('portfolio:positions.open')}
          </Button>
        ) : null,
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('portfolio:positions.empty')} />;
  }

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => `${r.exchange}:${r.symbol}`}
        size="small"
        pagination={false}
        loading={loading}
        columns={columns}
        dataSource={items}
        scroll={{ x: 720 }}
      />
    </DataTableCard>
  );
}
