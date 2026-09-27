import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './WallsTable.styles';
import type { WallRow, WallsTableProps } from './WallsTable.types';

function cell(value: string) {
  return (
    <Text variant="numeric" color="primary">
      {value || '—'}
    </Text>
  );
}

export function WallsTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: WallsTableProps) {
  const columns: ColumnsType<WallRow> = [
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
    { title: colLabels.venue, dataIndex: 'exchange', key: 'exchange', render: cell },
    { title: colLabels.side, dataIndex: 'side', key: 'side', render: cell },
    { title: colLabels.price, dataIndex: 'price', key: 'price', align: 'right', render: cell },
    { title: colLabels.notional, dataIndex: 'notional', key: 'notional', align: 'right', render: cell },
    { title: colLabels.distance, dataIndex: 'distance', key: 'distance', align: 'right', render: cell },
    { title: colLabels.kind, dataIndex: 'kind', key: 'kind', render: cell },
    { title: colLabels.appears, dataIndex: 'appears', key: 'appears', align: 'right', render: cell },
    { title: colLabels.present, dataIndex: 'present', key: 'present', render: cell },
  ];

  return (
    <TableCard>
      <StyledTable
        rowKey={(r) => `${r.exchange}:${r.symbol}:${r.side}:${r.price}:${r.kind}`}
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
        locale={{ emptyText: <DeskEmpty title={!hasLoaded ? emptyHint : emptyTitle} /> }}
        size="small"
        scroll={{ x: 960 }}
      />
    </TableCard>
  );
}
