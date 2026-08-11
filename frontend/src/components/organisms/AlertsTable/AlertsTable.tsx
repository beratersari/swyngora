import { Button, Popconfirm, Space } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import type { PriceAlert } from '@/libs/api/endpoints/alertsApi';
import { formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { AlertsTableProps } from './AlertsTable.types';

export function AlertsTable({
  items,
  loading,
  onDelete,
  onOpen,
  deleteLoading,
}: AlertsTableProps) {
  const { t } = useTranslation(['alerts', 'common']);

  const columns: ColumnsType<PriceAlert> = [
    {
      title: t('alerts:symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Text variant="label" mono color="primary">
          {formatSymbolDisplay(row.symbol)}
        </Text>
      ),
    },
    {
      title: t('alerts:exchange'),
      dataIndex: 'exchange',
      key: 'exchange',
      render: (ex: string | undefined) =>
        ex ? <BrandTag variant="exchange">{ex}</BrandTag> : <Text variant="caption">—</Text>,
    },
    {
      title: t('alerts:condition'),
      key: 'condition',
      render: (_, row) => (
        <Text variant="body">
          {row.condition === 'above' ? '≥' : '≤'} {row.targetPrice}
          {row.mode === 'repeating' ? ` (${t('alerts:modes.repeating', { defaultValue: 'repeating' })})` : ''}
        </Text>
      ),
    },
    {
      title: t('alerts:status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      render: (status: string | undefined) =>
        status ? (
          <BrandTag variant={status === 'triggered' ? 'live' : 'status'}>{status}</BrandTag>
        ) : (
          <Text variant="caption">—</Text>
        ),
    },
    {
      title: t('alerts:actions'),
      key: 'actions',
      render: (_, row) => (
        <Space>
          {onOpen && row.exchange && row.symbol ? (
            <Button size="small" type="link" onClick={() => onOpen(row.exchange!, row.symbol!)}>
              {t('alerts:open')}
            </Button>
          ) : null}
          <Popconfirm
            title={t('alerts:deleteConfirm', { defaultValue: 'Delete this alert?' })}
            okText={t('alerts:delete')}
            cancelText={t('common:actions.cancel', { defaultValue: 'Cancel' })}
            onConfirm={() => row.id && onDelete(row.id)}
          >
            <Button size="small" danger loading={deleteLoading}>
              {t('alerts:delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id ?? `${r.exchange}:${r.symbol}:${r.condition}`}
        loading={loading}
        dataSource={items}
        columns={columns}
        pagination={{ pageSize: 20 }}
        locale={{ emptyText: <DeskEmpty title={t('alerts:empty')} /> }}
        size="small"
        scroll={{ x: 640 }}
      />
    </DataTableCard>
  );
}
