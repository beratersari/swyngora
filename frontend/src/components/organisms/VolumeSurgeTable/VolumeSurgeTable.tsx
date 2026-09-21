import type { KeyboardEvent } from 'react';
import type { ColumnsType } from 'antd/es/table';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { Text } from '@/components/atoms/Text';
import { formatSymbolDisplay } from '@/libs/utils';
import { StyledTable, TableCard } from './VolumeSurgeTable.styles';
import type { VolumeSurgeRow, VolumeSurgeTableProps } from './VolumeSurgeTable.types';

export function VolumeSurgeTable({
  rows,
  loading = false,
  hasScanned,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
}: VolumeSurgeTableProps) {
  const columns: ColumnsType<VolumeSurgeRow> = [
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
      title: colLabels.maxRatio,
      dataIndex: 'maxRatio',
      key: 'maxRatio',
      align: 'right',
      render: (v: string) => (
        <Text variant="numeric" color="primary">
          {v ? `${v}×` : '—'}
        </Text>
      ),
    },
    {
      title: colLabels.hottest,
      dataIndex: 'hottest',
      key: 'hottest',
      render: (v: string) => (
        <Text variant="caption" color="secondary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.grade,
      dataIndex: 'grade',
      key: 'grade',
      render: (v: string) => (
        <Text variant="label" color="primary">
          {v || '—'}
        </Text>
      ),
    },
    {
      title: colLabels.dominant,
      dataIndex: 'dominant',
      key: 'dominant',
      render: (v: string) => (
        <Text variant="caption" color="secondary">
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
        dataSource={hasScanned ? rows : []}
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
          emptyText: <DeskEmpty title={!hasScanned ? emptyHint : emptyTitle} />,
        }}
        size="small"
        scroll={{ x: 720 }}
      />
    </TableCard>
  );
}
