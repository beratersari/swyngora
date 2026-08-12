import { Button, Popconfirm } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { PendingOrder } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioOrdersTableProps } from './PortfolioOrdersTable.types';

export function PortfolioOrdersTable({
  items,
  loading,
  cancelLoading,
  onCancel,
  onOpen,
}: PortfolioOrdersTableProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const columns: ColumnsType<PendingOrder> = [
    {
      title: t('portfolio:positions.symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Button
          type="link"
          size="small"
          disabled={!onOpen || !row.exchange || !row.symbol}
          onClick={() => row.exchange && row.symbol && onOpen?.(row.exchange, row.symbol)}
        >
          {formatSymbolDisplay(row.symbol)}
        </Button>
      ),
    },
    {
      title: t('portfolio:orders.type'),
      dataIndex: 'type',
      key: 'type',
      render: (v: string | undefined) => v ?? '—',
    },
    {
      title: t('portfolio:trade.side'),
      dataIndex: 'side',
      key: 'side',
      render: (v: string | undefined) =>
        v ? <BrandTag variant={v === 'buy' ? 'live' : 'status'}>{v}</BrandTag> : '—',
    },
    {
      title: t('portfolio:orders.trigger'),
      dataIndex: 'triggerPrice',
      key: 'trigger',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:orders.remaining'),
      dataIndex: 'remainingQuantity',
      key: 'rem',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:orders.status'),
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: t('portfolio:orders.cancel'),
      key: 'cancel',
      render: (_, row) =>
        onCancel && row.id && row.status === 'open' ? (
          <Popconfirm
            title={t('portfolio:orders.cancelConfirm')}
            onConfirm={() => onCancel(row.id!)}
            okText={t('portfolio:orders.cancel')}
            cancelText={t('common:actions.cancel', { defaultValue: 'Cancel' })}
          >
            <Button size="small" danger loading={cancelLoading}>
              {t('portfolio:orders.cancel')}
            </Button>
          </Popconfirm>
        ) : (
          <Text variant="caption">—</Text>
        ),
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('portfolio:orders.empty')} />;
  }

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id ?? `${r.symbol}-${r.createdAt}`}
        size="small"
        pagination={false}
        loading={loading}
        columns={columns}
        dataSource={items}
      />
    </DataTableCard>
  );
}
