import { useState } from 'react';
import { Button, InputNumber, Modal, Popconfirm, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { MarginPosition } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { semanticColors } from '@/styles/tokens';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import {
  marginBracketChange,
  type MarginBracketIssue,
} from './PortfolioMarginPositionsTable.helpers';
import type { PortfolioMarginPositionsTableProps } from './PortfolioMarginPositionsTable.types';

function pnlColor(n: number | undefined): string | undefined {
  if (n == null || !Number.isFinite(n) || Math.abs(n) < 1e-12) return undefined;
  return n > 0 ? semanticColors.chart.up : semanticColors.chart.down;
}

function asPrice(value: number | string | null): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }
  return null;
}

function isOpen(row: MarginPosition): boolean {
  return Boolean(row.id) && row.status === 'open';
}

export function PortfolioMarginPositionsTable({
  items,
  loading,
  closeLoading,
  bracketsLoading,
  onClose,
  onBrackets,
  onOpen,
}: PortfolioMarginPositionsTableProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [edit, setEdit] = useState<MarginPosition | null>(null);
  const [stopLoss, setStopLoss] = useState<number | null>(null);
  const [takeProfit, setTakeProfit] = useState<number | null>(null);
  const [issue, setIssue] = useState<MarginBracketIssue | null>(null);

  const openEdit = (row: MarginPosition) => {
    setEdit(row);
    setStopLoss(row.stopLoss ?? null);
    setTakeProfit(row.takeProfit ?? null);
    setIssue(null);
  };

  const columns: ColumnsType<MarginPosition> = [
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
      title: t('portfolio:margin.side'),
      dataIndex: 'side',
      key: 'side',
      render: (v: string | undefined) =>
        v ? <BrandTag variant={v === 'long' ? 'live' : 'status'}>{v}</BrandTag> : '—',
    },
    {
      title: t('portfolio:margin.leverage'),
      dataIndex: 'leverage',
      key: 'lev',
      align: 'right',
      render: (v: number | undefined) => (v != null ? `${v}x` : '—'),
    },
    {
      title: t('portfolio:positions.qty'),
      dataIndex: 'quantity',
      key: 'qty',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:margin.entry'),
      dataIndex: 'entryPrice',
      key: 'entry',
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
      title: t('portfolio:margin.liq'),
      dataIndex: 'liquidationPrice',
      key: 'liq',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:trade.stopLoss'),
      dataIndex: 'stopLoss',
      key: 'stop',
      align: 'right',
      render: (v: number | null | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:trade.takeProfit'),
      dataIndex: 'takeProfit',
      key: 'tp',
      align: 'right',
      render: (v: number | null | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:margin.margin'),
      dataIndex: 'margin',
      key: 'margin',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('portfolio:margin.debt'),
      key: 'debt',
      align: 'right',
      render: (_, row) => formatPrice((row.debtPrincipal ?? 0) + (row.debtInterest ?? 0)),
    },
    {
      title: t('portfolio:positions.uPnl'),
      dataIndex: 'unrealizedPnL',
      key: 'upnl',
      align: 'right',
      render: (v: number | undefined) => (
        <span style={{ color: pnlColor(v) }}>{formatPrice(v)}</span>
      ),
    },
    {
      title: t('portfolio:orders.actions'),
      key: 'actions',
      render: (_, row) => {
        const open = isOpen(row);
        if (!open || (!onClose && !onBrackets)) return '—';
        return (
          <Space size={4} wrap>
            {onBrackets ? (
              <Button size="small" onClick={() => openEdit(row)}>
                {t('portfolio:margin.editBrackets')}
              </Button>
            ) : null}
            {onClose ? (
              <Popconfirm
                title={t('portfolio:margin.closeConfirm')}
                onConfirm={() => void onClose(row.id!)}
                okText={t('portfolio:margin.close')}
                cancelText={t('common:actions.cancel', { defaultValue: 'Cancel' })}
              >
                <Button size="small" danger loading={closeLoading}>
                  {t('portfolio:margin.close')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        );
      },
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('portfolio:margin.emptyPositions')} />;
  }

  const openedStop = edit?.stopLoss ?? null;
  const openedTake = edit?.takeProfit ?? null;

  return (
    <>
      <DataTableCard>
        <DataTable
          rowKey={(r) => r.id ?? `${r.symbol}-${r.openedAt}`}
          size="small"
          pagination={false}
          loading={loading}
          columns={columns}
          dataSource={items}
          scroll={{ x: 1100 }}
        />
      </DataTableCard>
      <Modal
        open={Boolean(edit)}
        title={t('portfolio:margin.bracketsTitle')}
        okText={t('portfolio:margin.bracketsSave')}
        cancelText={t('common:actions.cancel', { defaultValue: 'Cancel' })}
        confirmLoading={bracketsLoading}
        onCancel={() => setEdit(null)}
        onOk={() => {
          if (!edit?.id || !onBrackets) return;
          const result = marginBracketChange({
            side: edit.side,
            entryPrice: edit.entryPrice,
            openedStop,
            openedTakeProfit: openedTake,
            stopLoss,
            takeProfit,
          });
          if ('issue' in result) {
            if (result.issue === 'unchanged') setEdit(null);
            else setIssue(result.issue);
            return;
          }
          void onBrackets({ id: edit.id, ...result.change })
            .then(() => setEdit(null))
            .catch(() => undefined);
        }}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Text variant="caption" color="secondary">
            {t('portfolio:margin.bracketsHint', { entry: formatPrice(edit?.entryPrice) })}
          </Text>
          <div>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.stopLoss')}
            </Text>
            <InputNumber
              aria-label={t('portfolio:trade.stopLoss')}
              style={{ width: '100%' }}
              value={stopLoss}
              onChange={(v) => {
                setStopLoss(asPrice(v));
                setIssue(null);
              }}
            />
            {stopLoss != null ? (
              <Button type="link" size="small" onClick={() => setStopLoss(null)}>
                {t('portfolio:margin.removeStop')}
              </Button>
            ) : openedStop != null ? (
              <Text variant="caption" color="secondary">
                {t('portfolio:margin.stopWillRemove')}
              </Text>
            ) : null}
          </div>
          <div>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.takeProfit')}
            </Text>
            <InputNumber
              aria-label={t('portfolio:trade.takeProfit')}
              style={{ width: '100%' }}
              value={takeProfit}
              onChange={(v) => {
                setTakeProfit(asPrice(v));
                setIssue(null);
              }}
            />
            {takeProfit != null ? (
              <Button type="link" size="small" onClick={() => setTakeProfit(null)}>
                {t('portfolio:margin.removeTakeProfit')}
              </Button>
            ) : openedTake != null ? (
              <Text variant="caption" color="secondary">
                {t('portfolio:margin.takeProfitWillRemove')}
              </Text>
            ) : null}
          </div>
          {issue === 'stop' || issue === 'takeProfit' ? (
            <Text variant="caption" color="error" role="alert">
              {issue === 'stop'
                ? t('portfolio:margin.badStop')
                : t('portfolio:margin.badTakeProfit')}
            </Text>
          ) : null}
        </Space>
      </Modal>
    </>
  );
}
