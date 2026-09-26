import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './PrintSizeTable.styles';
import type { PrintSizeRow, PrintSizeTableProps } from './PrintSizeTable.types';

function cell(value: string) {
  return (
    <Text variant="numeric" color="primary">
      {value || '—'}
    </Text>
  );
}

export function PrintSizeTable({
  rows,
  loading = false,
  hasLoaded,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: PrintSizeTableProps) {
  const columns: ColumnsType<PrintSizeRow> = [
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
    { title: colLabels.buyAvg, dataIndex: 'buyAvg', key: 'buyAvg', align: 'right', render: cell },
    { title: colLabels.buyPrior, dataIndex: 'buyPrior', key: 'buyPrior', align: 'right', render: cell },
    { title: colLabels.sellAvg, dataIndex: 'sellAvg', key: 'sellAvg', align: 'right', render: cell },
    { title: colLabels.sellPrior, dataIndex: 'sellPrior', key: 'sellPrior', align: 'right', render: cell },
    {
      title: colLabels.read,
      dataIndex: 'read',
      key: 'read',
      render: (v: string) => (
        <Text variant="label" color="primary">
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
