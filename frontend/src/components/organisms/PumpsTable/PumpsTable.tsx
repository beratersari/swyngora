import type { KeyboardEvent } from 'react';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { ColumnsType } from 'antd/es/table';
import { Text } from '@/components/atoms/Text';
import {
  changeTone,
  formatChangePercent,
  formatDateTime,
  formatSymbolDisplay,
} from '@/libs/utils';
import type { PumpScanRow } from '@/libs/utils/pumpScan';
import { StyledTable, TableCard } from './PumpsTable.styles';
import type { PumpsTableProps } from './PumpsTable.types';

export function PumpsTable({
  rows,
  loading = false,
  hasScanned,
  emptyHint,
  emptyTitle,
  columns: colLabels,
  onRowOpen,
  locale,
}: PumpsTableProps) {
  const columns: ColumnsType<PumpScanRow> = [
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
      title: colLabels.returnPct,
      dataIndex: 'returnPct',
      key: 'returnPct',
      align: 'right',
      render: (v: number | null) => (
        <Text variant="numeric" color={changeTone(v)}>
          {formatChangePercent(v)}
        </Text>
      ),
    },
    {
      title: colLabels.volumeRatio,
      dataIndex: 'volumeRatio',
      key: 'volumeRatio',
      align: 'right',
      render: (v: number | null) => (
        <Text variant="numeric">
          {v != null && Number.isFinite(v) ? v.toFixed(2) : '—'}
        </Text>
      ),
    },
    {
      title: colLabels.time,
      dataIndex: 'openTime',
      key: 'openTime',
      render: (v: string | null) => (
        <Text variant="caption" color="secondary">
          {formatDateTime(v, locale)}
        </Text>
      ),
    },
    {
      title: colLabels.events,
      dataIndex: 'eventCount',
      key: 'eventCount',
      align: 'right',
      render: (n: number) => <Text variant="numeric">{n}</Text>,
    },
  ];

  return (
    <TableCard>
      <StyledTable
        rowKey={(r) => `${r.exchange}:${r.symbol}:${r.openTime ?? r.returnPct}`}
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
        scroll={{ x: 640 }}
      />
    </TableCard>
  );
}
