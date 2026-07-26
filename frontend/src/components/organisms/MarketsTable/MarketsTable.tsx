import type { KeyboardEvent } from 'react';
import { Alert, Button, Empty, Space, Tag } from 'antd';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { FilterValue, SorterResult, TableCurrentDataSource } from 'antd/es/table/interface';
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
import {
  formatResultsRange,
  fromAntdSortOrder,
  paginationChanged,
  resolveTableChangeAction,
  toAntdSortOrder,
} from './MarketsTable.helpers';
import {
  EmptyWrap,
  SkeletonRow,
  StyledTable,
  TableCard,
  TableSkeletonWrap,
  TagList,
} from './MarketsTable.styles';
import type { MarketsTableProps } from './MarketsTable.types';

function MarketsTableSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <TableSkeletonWrap aria-label="Loading markets table" role="status">
      <SkeletonRow>
        {Array.from({ length: 7 }).map((_, i) => (
          <Skeleton key={`h-${i}`} variant="text" height={14} width="80%" active />
        ))}
      </SkeletonRow>
      {Array.from({ length: rows }).map((_, r) => (
        <SkeletonRow key={`r-${r}`}>
          {Array.from({ length: 7 }).map((_, c) => (
            <Skeleton key={`c-${r}-${c}`} variant="text" height={18} width="90%" active />
          ))}
        </SkeletonRow>
      ))}
    </TableSkeletonWrap>
  );
}

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
  onRowOpen,
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
                <Button size="small" type="primary" onClick={onRetry}>
                  Retry
                </Button>
              ) : undefined
            }
          />
        </EmptyWrap>
      </TableCard>
    );
  }

  // Initial load only — keep previous rows visible while paging/refetching
  if (isLoading && items.length === 0) {
    return (
      <TableCard>
        <MarketsTableSkeleton rows={Math.min(10, Math.max(6, limit > 20 ? 10 : 8))} />
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
        <Text variant="label" color="primary" mono>
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
    current: limit > 0 ? Math.floor(offset / limit) + 1 : 1,
    pageSize: limit,
    total,
    showSizeChanger: true,
    pageSizeOptions: [...PAGE_SIZE_OPTIONS],
    showTotal: () => formatResultsRange(offset, limit, total),
    // Keep pager usable with large totals
    showQuickJumper: total > limit * 5,
    hideOnSinglePage: false,
    position: ['bottomCenter'],
  };

  const handleChange = (
    pag: TablePaginationConfig,
    _filters: Record<string, FilterValue | null>,
    sorter: SorterResult<SpotMarket> | SorterResult<SpotMarket>[],
    extra: TableCurrentDataSource<SpotMarket>,
  ) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter;
    const field = s?.columnKey ? COLUMN_SORT[String(s.columnKey)] : undefined;
    const nextOrder = fromAntdSortOrder(s?.order ?? null);
    const sortChanged = Boolean(
      field && nextOrder && (field !== sort || nextOrder !== order),
    );
    const pageInfo = paginationChanged(pag, offset, limit);

    const action = resolveTableChangeAction(extra?.action, sortChanged, pageInfo.changed);

    if (action === 'sort') {
      if (field && nextOrder) {
        onSortChange(field as SpotSortField, nextOrder);
      }
      return;
    }

    if (action === 'paginate' || action === 'filter') {
      if (pageInfo.changed) {
        onPageChange(pageInfo.nextOffset, pageInfo.nextLimit);
      }
      return;
    }

    // Last resort: if page moved, page; if sort moved, sort
    if (pageInfo.changed) {
      onPageChange(pageInfo.nextOffset, pageInfo.nextLimit);
      return;
    }
    if (sortChanged && field && nextOrder) {
      onSortChange(field as SpotSortField, nextOrder);
    }
  };

  return (
    <TableCard>
      <StyledTable<SpotMarket>
        rowKey={(row, index) => row.symbol ?? `row-${index}`}
        columns={columns}
        dataSource={items}
        loading={isLoading && items.length > 0}
        pagination={pagination}
        onChange={handleChange}
        onRow={(record) => ({
          className: onRowOpen && record.symbol ? 'markets-row-clickable' : undefined,
          onClick: () => {
            if (onRowOpen && record.symbol) onRowOpen(record.symbol);
          },
          onKeyDown: (e: KeyboardEvent) => {
            if (!onRowOpen || !record.symbol) return;
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onRowOpen(record.symbol);
            }
          },
          tabIndex: onRowOpen && record.symbol ? 0 : undefined,
          role: onRowOpen ? 'link' : undefined,
          'aria-label': record.symbol ? `Open ${record.symbol} detail` : undefined,
        })}
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
