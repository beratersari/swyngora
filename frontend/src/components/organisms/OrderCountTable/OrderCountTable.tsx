import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './OrderCountTable.styles';
import type { OrderCountRow, OrderCountTableProps } from './OrderCountTable.types';

export function OrderCountTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: OrderCountTableProps) {
  const columns: ColumnsType<OrderCountRow> = [
    {
      title: colLabels.symbol,
      dataIndex: 'symbol',
      key: 'symbol',
      render: (s: string) => (
        <Text variant="label" mono color="primary">
          {formatSymbolDisplay(s)}
        </Text>
      ),
    },
    {
      title: colLabels.ratio,
      dataIndex: 'ratio',
      key: 'ratio',
      align: 'right',
      render: (v: string) => (
        <Text variant="numeric" color="primary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.lean,
      dataIndex: 'lean',
      key: 'lean',
      render: (v: string) => (
        <Text variant="label" color="primary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.buyCount,
      dataIndex: 'buyCount',
      key: 'buyCount',
      align: 'right',
      render: (v: string) => (
        <Text variant="numeric" color="primary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.sellCount,
      dataIndex: 'sellCount',
      key: 'sellCount',
      align: 'right',
      render: (v: string) => (
        <Text variant="numeric" color="primary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.summary,
      dataIndex: 'summary',
      key: 'summary',
      render: (v: string) => (
        <Text variant="caption" color="secondary">
          {v || '—'}
        </Text>
      ),
    },
  ];

  return (
    <TableCard>
      <StyledTable
        rowKey={(r) => `${r.exchange}:${r.symbol}`}
        loading={loading}
        dataSource={hasLoaded ? rows : []}
        columns={columns}
        pagination={{ pageSize: 20 }}
        onRow={(record) => ({
          onClick: () => onRowOpen(record.exchange, record.symbol),
          onKeyDown: (e: KeyboardEvent) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onRowOpen(record.exchange, record.symbol);
            }
          },
          tabIndex: 0,
          role: 'link',
          style: { cursor: 'pointer' },
        })}
        locale={{
          emptyText: <DeskEmpty title={!hasLoaded ? emptyHint : emptyTitle} />,
        }}
        size="small"
        scroll={{ x: 720 }}
      />
    </TableCard>
  );
}
