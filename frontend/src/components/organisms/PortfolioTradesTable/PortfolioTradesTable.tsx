import { Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { PaperTrade } from '@/libs/api';
import { formatDateTime, formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { PortfolioTradesTableProps } from './PortfolioTradesTable.types';

export function PortfolioTradesTable({ items, loading, onOpen }: PortfolioTradesTableProps) {
  const { t } = useTranslation('portfolio');
  const columns: ColumnsType<PaperTrade> = [
    {
      title: t('trades.time'),
      dataIndex: 'createdAt',
      key: 't',
      render: (v: string | undefined) => (
        <Text variant="caption" mono>
          {formatDateTime(v)}
        </Text>
      ),
    },
    {
      title: t('positions.symbol'),
      key: 'sym',
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
      title: t('trades.side'),
      dataIndex: 'side',
      key: 'side',
      render: (v: string | undefined) =>
        v ? <BrandTag variant={v === 'buy' ? 'live' : 'status'}>{v}</BrandTag> : '—',
    },
    {
      title: t('positions.qty'),
      dataIndex: 'quantity',
      key: 'q',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('trades.price'),
      dataIndex: 'price',
      key: 'p',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('trades.fee'),
      dataIndex: 'fee',
      key: 'f',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
    {
      title: t('trades.realized'),
      dataIndex: 'realizedPnL',
      key: 'r',
      align: 'right',
      render: (v: number | undefined) => formatPrice(v),
    },
  ];

  if (!loading && items.length === 0) {
    return <DeskEmpty title={t('trades.empty')} />;
  }

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id ?? `${r.symbol}-${r.createdAt}`}
        size="small"
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        loading={loading}
        columns={columns}
        dataSource={items}
      />
    </DataTableCard>
  );
}
