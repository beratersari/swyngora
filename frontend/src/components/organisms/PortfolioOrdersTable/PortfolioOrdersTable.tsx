import { useState } from 'react';
import { Button, InputNumber, Modal, Popconfirm, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { PendingOrder } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioOrdersTableProps } from './PortfolioOrdersTable.types';

function isAmendable(row: PendingOrder): boolean {
  if (!row.id || row.status !== 'open') return false;
  if (row.timeInForce && row.timeInForce !== 'gtc') return false;
  if (row.ocoGroupId || row.bracketId || row.type === 'trailing_stop') return false;
  return row.type === 'limit_buy' || row.type === 'limit_sell' || row.type === 'stop_loss';
}

export function PortfolioOrdersTable({
  items,
  loading,
  cancelLoading,
  amendLoading,
  onCancel,
  onAmend,
  onOpen,
}: PortfolioOrdersTableProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [edit, setEdit] = useState<PendingOrder | null>(null);
  const [trigger, setTrigger] = useState<number | null>(null);
  const [remaining, setRemaining] = useState<number | null>(null);

  const openAmend = (row: PendingOrder) => {
    setEdit(row);
    setTrigger(row.triggerPrice ?? null);
    setRemaining(row.remainingQuantity ?? null);
  };

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
      title: t('portfolio:orders.filled'),
      dataIndex: 'filledQuantity',
      key: 'filled',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v ?? 0),
    },
    {
      title: t('portfolio:orders.remaining'),
      dataIndex: 'remainingQuantity',
      key: 'rem',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:orders.tif'),
      dataIndex: 'timeInForce',
      key: 'tif',
      render: (v: string | undefined) => (v ? v.toUpperCase() : '—'),
    },
    {
      title: t('portfolio:orders.status'),
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: t('portfolio:orders.actions'),
      key: 'actions',
      render: (_, row) => (
        <Space size={4} wrap>
          {onAmend && isAmendable(row) ? (
            <Button size="small" onClick={() => openAmend(row)}>
              {t('portfolio:orders.amend')}
            </Button>
          ) : null}
          {onCancel && row.id && (row.status === 'open' || row.status === 'pending') ? (
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
            !onAmend || !isAmendable(row) ? <Text variant="caption">—</Text> : null
          )}
        </Space>
      ),
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('portfolio:orders.empty')} />;
  }

  return (
    <>
      <DataTableCard>
        <DataTable
          rowKey={(r) => r.id ?? `${r.symbol}-${r.createdAt}`}
          size="small"
          pagination={false}
          loading={loading}
          columns={columns}
          dataSource={items}
          scroll={{ x: 720 }}
        />
      </DataTableCard>
      <Modal
        open={Boolean(edit)}
        title={t('portfolio:orders.amendTitle')}
        okText={t('portfolio:orders.amend')}
        confirmLoading={amendLoading}
        onCancel={() => setEdit(null)}
        onOk={() => {
          if (!edit?.id || !onAmend) return;
          void onAmend({
            id: edit.id,
            triggerPrice: trigger ?? undefined,
            remainingQuantity: remaining ?? undefined,
          }).then(() => setEdit(null));
        }}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <div>
            <Text variant="caption" color="secondary">
              {t('portfolio:orders.trigger')}
            </Text>
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              value={trigger}
              onChange={(v) => setTrigger(typeof v === 'number' ? v : null)}
            />
          </div>
          <div>
            <Text variant="caption" color="secondary">
              {t('portfolio:orders.remaining')}
            </Text>
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              value={remaining}
              onChange={(v) => setRemaining(typeof v === 'number' ? v : null)}
            />
          </div>
        </Space>
      </Modal>
    </>
  );
}
