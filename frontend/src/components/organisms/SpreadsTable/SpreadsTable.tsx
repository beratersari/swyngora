import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './SpreadsTable.styles';
import type { SpreadRow, SpreadsTableProps } from './SpreadsTable.types';

function cell(value: string) {
  return (
    <Text variant="numeric" color="primary">
      {value || '—'}
    </Text>
  );
}

export function SpreadsTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: SpreadsTableProps) {
  const columns: ColumnsType<SpreadRow> = [
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
    { title: colLabels.exchange, dataIndex: 'exchange', key: 'exchange', render: cell },
    { title: colLabels.current, dataIndex: 'current', key: 'current', align: 'right', render: cell },
    { title: colLabels.typical, dataIndex: 'typical', key: 'typical', align: 'right', render: cell },
    { title: colLabels.ratio, dataIndex: 'ratio', key: 'ratio', align: 'right', render: cell },
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
        scroll={{ x: 720 }}
      />
    </TableCard>
  );
}
