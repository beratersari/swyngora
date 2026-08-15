import type { KeyboardEvent, MouseEvent } from 'react';
import { useCallback, useMemo } from 'react';
import { Alert, Button } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { WatchStar } from '@/components/molecules/WatchStar';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { FilterValue, SorterResult, TableCurrentDataSource } from 'antd/es/table/interface';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { SpotMarket, SpotSortField } from '@/libs/api';
import { SpotMetricValue } from '@/components/molecules/SpotMetricValue';
import {
  defaultMetricIds,
  formatSymbolDisplay,
  metricColumnTitle,
  resolveMetricDefs,
} from '@/libs/utils';
import { COLUMN_SORT, PAGE_SIZE_OPTIONS, SORT_DIRECTIONS } from './MarketsTable.constants';
import {
  getResultsRange,
  paginationChanged,
  resolveSortChange,
  resolveTableChangeAction,
  toAntdSortOrder,
} from './MarketsTable.helpers';
import {
  EmptyWrap,
  NameCell,
  SkeletonRow,
  StyledTable,
  TableCard,
  TableSkeletonWrap,
} from './MarketsTable.styles';
import type { MarketsTableProps } from './MarketsTable.types';

function MarketsTableSkeleton({ rows = 8, label }: { rows?: number; label: string }) {
  return (
    <TableSkeletonWrap aria-label={label} role="status">
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
  watchedKeys,
  onToggleWatch,
  metrics: metricsProp,
}: MarketsTableProps) {
  const { t } = useTranslation(['markets', 'common', 'watchlist']);

  const metrics = useMemo(
    () => metricsProp ?? resolveMetricDefs('markets', defaultMetricIds('markets')),
    [metricsProp],
  );

  // Server-side sort only (sorter: true). Never allow Ant Design's third-click
  // "clear sort" — that sets order=null while our URL still holds a sort, which
  // fights controlled sortOrder and makes the header feel unstable.
  // Metric columns come from the shared catalog + column picker selection.
  const columns: ColumnsType<SpotMarket> = useMemo(() => {
    const watchCol: ColumnsType<SpotMarket> = onToggleWatch
      ? [
          {
            title: '',
            key: 'watch',
            width: 44,
            render: (_: unknown, row: SpotMarket) => {
              const sym = row.symbol ?? '';
              const key = `${exchange}:${sym}`;
              const watched = watchedKeys?.has(key) ?? false;
              return (
                <WatchStar
                  watched={watched}
                  addLabel={t('watchlist:add')}
                  removeLabel={t('watchlist:remove')}
                  onClick={(e: MouseEvent) => {
                    e.stopPropagation();
                    if (sym) onToggleWatch(sym, watched);
                  }}
                />
              );
            },
          },
        ]
      : [];

    const metricCols: ColumnsType<SpotMarket> = metrics.map((def) => {
      const sortable = Boolean(def.sortField);
      return {
        title: metricColumnTitle(t, def.labelKey),
        dataIndex: def.field,
        key: def.id,
        align: def.align ?? 'right',
        ...(sortable
          ? {
              sorter: true as const,
              sortDirections: [...SORT_DIRECTIONS],
              sortOrder: toAntdSortOrder(order, sort === def.sortField),
            }
          : {}),
        render: (_: unknown, row: SpotMarket) => (
          <SpotMetricValue metric={def} spot={row} exchange={exchange} />
        ),
      };
    });

    return [
      ...watchCol,
      {
        title: t('markets:table.symbol'),
        dataIndex: 'symbol',
        key: 'symbol',
        sorter: true,
        sortDirections: [...SORT_DIRECTIONS],
        sortOrder: toAntdSortOrder(order, sort === 'symbol'),
        render: (symbol: string | undefined) => (
          <NameCell>
            <Text variant="label" color="primary">
              {formatSymbolDisplay(symbol)}
            </Text>
          </NameCell>
        ),
      },
      ...metricCols,
    ];
  }, [exchange, metrics, onToggleWatch, order, sort, t, watchedKeys]);

  const range = getResultsRange(offset, limit, total);
  const pagination: TablePaginationConfig = {
    current: limit > 0 ? Math.floor(offset / limit) + 1 : 1,
    pageSize: limit,
    total,
    showSizeChanger: true,
    pageSizeOptions: [...PAGE_SIZE_OPTIONS],
    showTotal: () =>
      range.kind === 'empty'
        ? t('markets:results.emptyMatches')
        : t('markets:results.range', {
            from: range.from.toLocaleString(),
            to: range.to.toLocaleString(),
            total: range.total.toLocaleString(),
          }),
    showQuickJumper: total > limit * 5,
    hideOnSinglePage: false,
    position: ['bottomCenter'],
  };

  const handleChange = useCallback((
    pag: TablePaginationConfig,
    _filters: Record<string, FilterValue | null>,
    sorter: SorterResult<SpotMarket> | SorterResult<SpotMarket>[],
    extra: TableCurrentDataSource<SpotMarket>,
  ) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter;
    const sortResult = resolveSortChange({
      columnKey: s?.columnKey as string | number | undefined,
      field: s?.field as string | number | undefined,
      antdOrder: s?.order ?? null,
      action: extra?.action,
      activeSort: sort,
      activeOrder: order,
      columnSortMap: COLUMN_SORT,
    });
    const sortChanged = sortResult.type === 'sort';
    const pageInfo = paginationChanged(pag, offset, limit);

    const action = resolveTableChangeAction(extra?.action, sortChanged, pageInfo.changed);

    if (action === 'sort') {
      if (sortResult.type === 'sort') {
        onSortChange(sortResult.field as SpotSortField, sortResult.order);
      }
      return;
    }

    if (action === 'paginate' || action === 'filter') {
      if (pageInfo.changed) {
        onPageChange(pageInfo.nextOffset, pageInfo.nextLimit);
      }
      return;
    }

    if (pageInfo.changed) {
      onPageChange(pageInfo.nextOffset, pageInfo.nextLimit);
      return;
    }
    if (sortResult.type === 'sort') {
      onSortChange(sortResult.field as SpotSortField, sortResult.order);
    }
  }, [limit, offset, onPageChange, onSortChange, order, sort]);

  if (errorMessage) {
    return (
      <TableCard>
        <EmptyWrap>
          <Alert
            type="error"
            showIcon
            message={t('markets:table.loadErrorTitle')}
            description={errorMessage}
            action={
              onRetry ? (
                <Button size="small" type="primary" onClick={onRetry}>
                  {t('common:actions.retry')}
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
        <MarketsTableSkeleton
          rows={Math.min(10, Math.max(6, limit > 20 ? 10 : 8))}
          label={t('common:a11y.loadingMarketsTable')}
        />
      </TableCard>
    );
  }

  return (
    <TableCard>
      <StyledTable<SpotMarket>
        rowKey={(row, index) => row.symbol ?? `row-${index}`}
        columns={columns}
        dataSource={items}
        loading={isLoading && items.length > 0}
        pagination={pagination}
        // Table-level directions (same as columns) — avoid antd default cycle ending in clear.
        sortDirections={[...SORT_DIRECTIONS]}
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
          'aria-label': record.symbol
            ? t('markets:table.openDetail', { symbol: formatSymbolDisplay(record.symbol) })
            : undefined,
        })}
        locale={{
          emptyText: (
            <DeskEmpty
              title={t('markets:table.emptyTitle')}
              hint={t('markets:table.emptyHint')}
            />
          ),
        }}
        size="small"
        scroll={{ x: 900 }}
      />
    </TableCard>
  );
}
