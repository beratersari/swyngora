import type { MouseEvent } from 'react';
import { useMemo } from 'react';
import { Button, Empty } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { WatchlistItem } from '@/libs/api';
import { defaultMetricIds, resolveMetricDefs } from '@/libs/utils';
import { WatchlistMetricCell, WatchlistPairCell } from './WatchlistSpotCells';
import { StyledTable, TableCard } from './WatchlistTable.styles';
import type { WatchlistTableProps } from './WatchlistTable.types';

export function WatchlistTable({
  items,
  loading,
  onRemove,
  removeLoading,
  onOpen,
  metrics: metricsProp,
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
      render: (_, row) => (row.symbol ? <WatchlistPairCell symbol={row.symbol} /> : '—'),
    },
    {
      title: t('watchlist:exchange'),
      dataIndex: 'exchange',
      key: 'exchange',
      render: (ex: string | undefined) => <Text variant="label">{ex ?? '—'}</Text>,
    },
    ...metrics.map((def) => ({
      title: t(`markets:table.${def.labelKey}`),
      key: def.id,
      align: (def.align ?? 'right') as 'left' | 'right',
      render: (_: unknown, row: WatchlistItem) =>
        row.symbol ? (
          <WatchlistMetricCell
            exchange={row.exchange ?? 'binance'}
            symbol={row.symbol}
            metric={def}
          />
        ) : (
          '—'
        ),
    })),
    {
      title: '',
      key: 'actions',
      width: 48,
      align: 'center',
      render: (_, row) => (
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
