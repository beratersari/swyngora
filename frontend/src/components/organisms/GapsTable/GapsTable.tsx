import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './GapsTable.styles';
import type { GapRow, GapsTableProps } from './GapsTable.types';

function cell(value: string) {
  return (
    <Text variant="numeric" color="primary">
      {value || '—'}
    </Text>
  );
}

export function GapsTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: GapsTableProps) {
  const columns: ColumnsType<GapRow> = [
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
    { title: colLabels.easier, dataIndex: 'easier', key: 'easier', render: cell },
    { title: colLabels.above, dataIndex: 'above', key: 'above', align: 'right', render: cell },
    { title: colLabels.below, dataIndex: 'below', key: 'below', align: 'right', render: cell },
    { title: colLabels.emptyAbove, dataIndex: 'emptyAbove', key: 'emptyAbove', align: 'right', render: cell },
    { title: colLabels.emptyBelow, dataIndex: 'emptyBelow', key: 'emptyBelow', align: 'right', render: cell },
    { title: colLabels.widest, dataIndex: 'widest', key: 'widest', align: 'right', render: cell },
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
        locale={{ emptyText: <DeskEmpty title={!hasLoaded ? emptyHint : emptyTitle} /> }}
        size="small"
        scroll={{ x: 880 }}
      />
    </TableCard>
  );
}
