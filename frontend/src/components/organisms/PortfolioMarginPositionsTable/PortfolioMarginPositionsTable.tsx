import { Button, Popconfirm } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { MarginPosition } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioMarginPositionsTableProps } from './PortfolioMarginPositionsTable.types';

function pnlColor(n: number | undefined): string | undefined {
  if (n == null || !Number.isFinite(n) || Math.abs(n) < 1e-12) return undefined;
  return n > 0 ? '#4FD4A5' : '#ff6b6b';
}

export function PortfolioMarginPositionsTable({
  items,
  loading,
  closeLoading,
  onClose,
  onOpen,
}: PortfolioMarginPositionsTableProps) {
  const { t } = useTranslation(['portfolio', 'common']);
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
      render: (_, row) =>
        formatPrice((row.debtPrincipal ?? 0) + (row.debtInterest ?? 0)),
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
      title: t('portfolio:margin.close'),
      key: 'close',
      render: (_, row) =>
        onClose && row.id && row.status === 'open' ? (
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
        ) : (
          '—'
        ),
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('portfolio:margin.emptyPositions')} />;
  }

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id ?? `${r.symbol}-${r.openedAt}`}
        size="small"
        pagination={false}
        loading={loading}
        columns={columns}
        dataSource={items}
      />
    </DataTableCard>
  );
}
