import type { KeyboardEvent, MouseEvent } from 'react';
import { useMemo } from 'react';
import { Button, Empty } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { WatchlistItem } from '@/libs/api';
import {
  defaultMetricIds,
  formatSymbolDisplay,
  metricColumnTitle,
  resolveMetricDefs,
} from '@/libs/utils';
import { StyledTable, TableCard } from './WatchlistTable.styles';
import type { WatchlistTableProps } from './WatchlistTable.types';

export function WatchlistTable({
  items,
  loading,
  onRemove,
  removeLoading,
  onOpen,
  metrics: metricsProp,
  renderMetric,
}: WatchlistTableProps) {
  const { t } = useTranslation(['watchlist', 'markets', 'common']);

  const metrics = useMemo(
    () => metricsProp ?? resolveMetricDefs('watchlist', defaultMetricIds('watchlist')),
    [metricsProp],
  );

  const columns: ColumnsType<WatchlistItem> = [
    {
      title: t('watchlist:symbol'),
      key: 'symbol',
      render: (_, row) =>
        row.symbol ? (
          <Text variant="label" mono color="primary">
            {formatSymbolDisplay(row.symbol)}
          </Text>
        ) : (
          '—'
        ),
    },
    {
      title: t('watchlist:exchange'),
      dataIndex: 'exchange',
      key: 'exchange',
      render: (ex: string | undefined) => <Text variant="label">{ex ?? '—'}</Text>,
    },
    ...metrics.map((def) => ({
      title: metricColumnTitle(t, def.labelKey),
      key: def.id,
      align: (def.align ?? 'right') as 'left' | 'right',
      render: (_: unknown, row: WatchlistItem) => {
        if (!row.symbol) return '—';
        if (!renderMetric) return '—';
        return renderMetric({
          exchange: row.exchange ?? 'binance',
          symbol: row.symbol,
          metric: def,
        });
      },
    })),
    {
      title: '',
      key: 'actions',
      width: 48,
      align: 'center' as const,
      render: (_: unknown, row: WatchlistItem) => (
        <Button
          type="text"
          size="small"
          danger
          loading={removeLoading}
          icon={<DeleteOutlined />}
          aria-label={t('watchlist:remove')}
          onClick={(e: MouseEvent) => {
            e.stopPropagation();
            if (row.exchange && row.symbol) onRemove(row.exchange, row.symbol);
          }}
        />
      ),
    },
  ];

  return (
    <TableCard>
      <StyledTable
        rowKey={(r) => `${r.exchange}:${r.symbol}`}
        loading={loading}
        dataSource={items}
        columns={columns}
        pagination={false}
        onRow={(record) => ({
          onClick: () => {
            if (record.exchange && record.symbol) {
              onOpen(record.exchange, record.symbol);
            }
          },
          onKeyDown: (e: KeyboardEvent) => {
            if (!record.exchange || !record.symbol) return;
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onOpen(record.exchange, record.symbol);
            }
          },
          tabIndex: record.symbol ? 0 : undefined,
          role: record.symbol ? 'link' : undefined,
          'aria-label':
            record.symbol != null
              ? t('watchlist:open', { symbol: record.symbol })
              : undefined,
          style: { cursor: record.symbol ? 'pointer' : undefined },
        })}
        locale={{
          emptyText: (
            <Empty description={t('watchlist:emptyTitle')}>
              <Text variant="caption" color="secondary">
                {t('watchlist:emptyHint')}
              </Text>
            </Empty>
          ),
        }}
      />
    </TableCard>
  );
}