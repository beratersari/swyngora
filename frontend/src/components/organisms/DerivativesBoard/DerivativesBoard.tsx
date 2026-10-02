import { Table } from 'antd';
import type { ColumnsType, TableProps } from 'antd/es/table';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { DerivativesCoin, DerivativesVenue } from '@/libs/api';
import type { BoardSort, BoardSortKey } from './DerivativesBoard.types';
import {
  coinChartPath,
  definiteText,
  fundingText,
  isBoardSortKey,
  percentText,
  plainAmount,
  priceText,
  signedAmount,
  sortBoardCoins,
} from './helpers';
import type { DerivativesBoardProps } from './DerivativesBoard.types';

const NUMERIC_KEYS: BoardSortKey[] = [
  'price',
  'funding',
  'openInterest',
  'change1h',
  'change24h',
  'liqLong',
  'liqShort',
  'longPct',
  'takerBuyPct',
];

export function DerivativesBoard({ exchange, coins, loading }: DerivativesBoardProps) {
  const { t } = useTranslation(['derivatives', 'common']);
  const navigate = useNavigate();
  const [sort, setSort] = useState<BoardSort | null>(null);
  const rows = sort ? sortBoardCoins(coins, sort) : coins;
  const venueLine = (venues: DerivativesVenue[] | undefined): string => {
    if (!venues?.length) return '';
    return venues
      .map((venue) => {
        const name = t(`common:exchanges.${venue.exchange ?? ''}`, {
          defaultValue: venue.exchange ?? '',
        });
        const status = t(`common:coverage.${venue.oiStatus ?? 'missing'}`, {
          defaultValue: venue.oiStatus ?? '',
        });
        return `${name} ${status}`;
      })
      .join(' · ');
  };
  const sortOrder = (key: BoardSortKey): 'ascend' | 'descend' | null => {
    if (sort?.key !== key) return null;
    return sort.dir === 'asc' ? 'ascend' : 'descend';
  };
  const sortable = (key: BoardSortKey) => ({
    key,
    sorter: true as const,
    sortOrder: sortOrder(key),
    sortDirections: (NUMERIC_KEYS.includes(key) ? ['descend', 'ascend'] : ['ascend', 'descend']) as (
      | 'ascend'
      | 'descend'
    )[],
  });
  const columns: ColumnsType<DerivativesCoin> = [
    {
      title: t('derivatives:coin'),
      dataIndex: 'symbol',
      fixed: 'left',
      ...sortable('base'),
      render: (_value, row) => (
        <Link to={coinChartPath(exchange, row.symbol ?? '')}>{row.base || row.symbol}</Link>
      ),
    },
    {
      title: t('derivatives:price'),
      dataIndex: 'price',
      ...sortable('price'),
      render: (_value, row) => priceText(row),
    },
    {
      title: t('derivatives:funding'),
      dataIndex: 'fundingRatePct',
      ...sortable('funding'),
      render: (_value, row) => fundingText(row),
    },
    {
      title: t('derivatives:openInterest'),
      dataIndex: 'openInterest',
      ...sortable('openInterest'),
      render: (_value, row) => definiteText(row.openInterestDefinite, row.openInterest, plainAmount),
    },
    {
      title: t('derivatives:change1h'),
      dataIndex: 'change1h',
      ...sortable('change1h'),
      render: (_value, row) => definiteText(row.change1hDefinite, row.change1h, signedAmount),
    },
    {
      title: t('derivatives:change24h'),
      dataIndex: 'change24h',
      ...sortable('change24h'),
      render: (_value, row) => definiteText(row.change24hDefinite, row.change24h, signedAmount),
    },
    {
      title: t('derivatives:longPct'),
      dataIndex: 'longPct',
      ...sortable('longPct'),
      render: (_value, row) => percentText(row.longDefinite, row.longPct),
    },
    {
      title: t('derivatives:takerBuy'),
      dataIndex: 'takerBuyPct',
      ...sortable('takerBuyPct'),
      render: (_value, row) => percentText(row.takerDefinite, row.takerBuyPct),
    },
    {
      title: t('derivatives:liqLong'),
      dataIndex: 'liqLong',
      ...sortable('liqLong'),
      render: (_value, row) => definiteText(row.liqDefinite, row.liqLong, plainAmount),
    },
    {
      title: t('derivatives:liqShort'),
      dataIndex: 'liqShort',
      ...sortable('liqShort'),
      render: (_value, row) => definiteText(row.liqDefinite, row.liqShort, plainAmount),
    },
    {
      title: t('derivatives:venues'),
      dataIndex: 'venues',
      render: (_value, row) => (
        <Text variant="caption" color="secondary">
          {venueLine(row.venues)}
        </Text>
      ),
    },
  ];
  const onChange: TableProps<DerivativesCoin>['onChange'] = (_pagination, _filters, sorter) => {
    const current = Array.isArray(sorter) ? sorter[0] : sorter;
    const key = current?.columnKey;
    if (!current?.order || typeof key !== 'string' || !isBoardSortKey(key)) {
      setSort(null);
      return;
    }
    setSort({ key, dir: current.order === 'ascend' ? 'asc' : 'desc' });
  };

  return (
    <Table<DerivativesCoin>
      rowKey={(row) => row.symbol ?? ''}
      columns={columns}
      dataSource={rows}
      loading={loading}
      pagination={false}
      scroll={{ x: 1280 }}
      locale={{ emptyText: t('derivatives:empty') }}
      onChange={onChange}
      onRow={(row) => ({
        onClick: (event) => {
          const target = event.target;
          if (target instanceof Element && target.closest('a')) return;
          if (!row.symbol) return;
          navigate(coinChartPath(exchange, row.symbol));
        },
        style: { cursor: 'pointer' },
      })}
    />
  );
}
