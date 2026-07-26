import { Alert, Button, Empty, Space, Tag } from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { FilterValue, SorterResult } from 'antd/es/table/interface';
import { Text } from '@/components/atoms/Text';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { SpotMarket, SpotSortField } from '@/libs/api';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
  formatTradeCount,
} from '@/libs/utils';
import { COLUMN_SORT, PAGE_SIZE_OPTIONS } from './MarketsTable.constants';
import { fromAntdSortOrder, toAntdSortOrder } from './MarketsTable.helpers';
import { EmptyWrap, StyledTable, TableCard, TagList } from './MarketsTable.styles';
import type { MarketsTableProps } from './MarketsTable.types';

export function MarketsTable({
  items,
  exchange,
  sort,
  order,
  total,
  limit,
  offset,
  isLoading,
  errorMessage,
  onSortChange,
  onPageChange,
  onRetry,
}: MarketsTableProps) {
  if (errorMessage) {
    return (
      <TableCard>
        <EmptyWrap>
          <Alert
            type="error"
            showIcon
            message="Could not load markets"
            description={errorMessage}
            action={
              onRetry ? (
                <Button size="small" onClick={onRetry}>
                  Retry
                </Button>
              ) : undefined
            }
          />
        </EmptyWrap>
      </TableCard>
    );
  }

  if (isLoading && items.length === 0) {
    return (
      <TableCard>
        <EmptyWrap>
          <Skeleton variant="card" height={320} active aria-label="Loading markets table" />
        </EmptyWrap>
      </TableCard>
    );
  }

  const columns: ColumnsType<SpotMarket> = [
    {
      title: 'Symbol',
      dataIndex: 'symbol',
      key: 'symbol',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'symbol'),
      render: (symbol: string | undefined) => (
        <Text variant="label" color="cream" mono>
          {symbol ?? '—'}
        </Text>
      ),
    },
    {
      title: 'Last',
      dataIndex: 'lastPrice',
      key: 'lastPrice',
      align: 'right',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'lastPrice'),
      render: (v: string | undefined) => (
        <Text variant="numeric" color="primary">
          {formatPrice(v)}
        </Text>
      ),
    },
    {
      title: '24h %',
      dataIndex: 'priceChangePercent',
      key: 'priceChangePercent',
      align: 'right',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'priceChangePercent'),
      render: (v: string | undefined) => (
        <Text variant="numeric" color={changeTone(v)}>
          {formatChangePercent(v)}
        </Text>
      ),
    },
    {
      title: 'Quote vol',
      dataIndex: 'quoteVolume',
      key: 'quoteVolume',
      align: 'right',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'quoteVolume'),
      render: (v: string | undefined) => (
        <Text variant="numeric" color="secondary">
          {formatCompactUsd(v)}
        </Text>
      ),
    },
    {
      title: 'Circ. mcap',
      dataIndex: 'marketCapCirculating',
      key: 'marketCapCirculating',
      align: 'right',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'marketCapCirculating'),
      render: (v: number | null | undefined) => (
        <Text variant="numeric" color="secondary">
          {formatCompactUsd(v)}
        </Text>
      ),
    },
    {
      title: 'Trades',
      dataIndex: 'tradeCount',
      key: 'tradeCount',
      align: 'right',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'tradeCount'),
      render: (v: number | undefined) => (
        <Text variant="numeric" color="secondary">
          {formatTradeCount(v, exchange)}
        </Text>
      ),
    },
    {
      title: 'Tags',
      dataIndex: 'tags',
      key: 'tags',
      sorter: true,
      sortOrder: toAntdSortOrder(order, sort === 'tags'),
      render: (tags: string[] | undefined) =>
        tags && tags.length > 0 ? (
          <TagList>
            {tags.slice(0, 4).map((t) => (
              <Tag key={t}>{t}</Tag>
            ))}
            {tags.length > 4 ? <Tag>+{tags.length - 4}</Tag> : null}
          </TagList>
        ) : (
          <Text variant="caption" color="secondary">
            —
          </Text>
        ),
    },
  ];

  const pagination: TablePaginationConfig = {
    current: Math.floor(offset / limit) + 1,
    pageSize: limit,
    total,
    showSizeChanger: true,
    pageSizeOptions: [...PAGE_SIZE_OPTIONS],
    showTotal: (t) => `${t.toLocaleString()} markets`,
  };

  const handleChange = (
    pag: TablePaginationConfig,
    _filters: Record<string, FilterValue | null>,
    sorter: SorterResult<SpotMarket> | SorterResult<SpotMarket>[],
  ) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter;
    if (s?.columnKey && s.order) {
      const field = COLUMN_SORT[String(s.columnKey)];
      const nextOrder = fromAntdSortOrder(s.order);
      if (field && nextOrder) {
        onSortChange(field as SpotSortField, nextOrder);
        return;
      }
    }

    const pageSize = pag.pageSize ?? limit;
    const page = pag.current ?? 1;
    onPageChange((page - 1) * pageSize, pageSize);
  };

  return (
    <TableCard>
      <StyledTable<SpotMarket>
        rowKey={(row) => row.symbol ?? String(Math.random())}
        columns={columns}
        dataSource={items}
        loading={isLoading && items.length > 0}
        pagination={pagination}
        onChange={handleChange}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <Space direction="vertical" size={4}>
                  <Text variant="body" color="secondary">
                    No markets match filters
                  </Text>
                  <Text variant="caption" color="secondary">
                    Try another exchange, quote, tag, or search
                  </Text>
                </Space>
              }
            />
          ),
        }}
        size="middle"
        scroll={{ x: 900 }}
      />
    </TableCard>
  );
}
