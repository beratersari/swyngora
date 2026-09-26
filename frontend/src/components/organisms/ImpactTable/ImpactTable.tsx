import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './ImpactTable.styles';
import type { ImpactRow, ImpactTableProps } from './ImpactTable.types';

function cell(value: string) {
  return (
    <Text variant="numeric" color="primary">
      {value || '—'}
    </Text>
  );
}

export function ImpactTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: ImpactTableProps) {
  const columns: ColumnsType<ImpactRow> = [
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
    { title: colLabels.slippage, dataIndex: 'slippage', key: 'slippage', align: 'right', render: cell },
    { title: colLabels.versusBest, dataIndex: 'versusBest', key: 'versusBest', align: 'right', render: cell },
    { title: colLabels.impact, dataIndex: 'impact', key: 'impact', align: 'right', render: cell },
    { title: colLabels.exhausted, dataIndex: 'exhausted', key: 'exhausted', render: cell },
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
