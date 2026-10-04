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
  formatRecordedSpan,
  fundingText,
  recordedSummary,
  recordedTitle,
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
  'gapPct',
  'cvdDay',
  'printChange',
  'liqHeat',
  'bidPct',
];

export function DerivativesBoard({ exchange, coins, loading }: DerivativesBoardProps) {
  const { t } = useTranslation(['derivatives', 'common']);
  const navigate = useNavigate();
  const venueName = (id: string): string => {
    switch (id) {
      case 'binance':
        return t('common:exchanges.binance');
      case 'bybit':
        return t('common:exchanges.bybit');
      case 'okx':
        return t('common:exchanges.okx');
      default:
        return id;
    }
  };
  const gapTitle = (row: DerivativesCoin): string | undefined => {
    if (!row.gapDefinite || !row.gapScope) return undefined;
    const names = row.gapScope.split(',').filter(Boolean).map(venueName);
    return t('derivatives:gapScope', { venues: names.join(', ') });
  };
  const printWord = (read: string): string => {
    switch (read) {
      case 'jumped':
        return t('derivatives:printRead.jumped');
      case 'faded':
        return t('derivatives:printRead.faded');
      case 'larger':
        return t('derivatives:printRead.larger');
      case 'smaller':
        return t('derivatives:printRead.smaller');
      case 'few_large':
        return t('derivatives:printRead.few_large');
      case 'many_small':
        return t('derivatives:printRead.many_small');
      default:
        return '';
    }
  };
  const printLabel = (row: DerivativesCoin): string => {
    if (!row.printDefinite || !row.printChangePct) return '—';
    const pct = `${row.printChangePct}%`;
    const word = row.printRead ? printWord(row.printRead) : '';
    return word ? `${pct} ${word}` : pct;
  };
  const heatWord = (grade: string): string => {
    switch (grade) {
      case 'quiet':
        return t('derivatives:heat.quiet');
      case 'elevated':
        return t('derivatives:heat.elevated');
      case 'cascade':
        return t('derivatives:heat.cascade');
      case 'extreme':
        return t('derivatives:heat.extreme');
      default:
        return grade;
    }
  };
  const heatLabel = (row: DerivativesCoin): string => {
    if (!row.liqHeatDefinite || !row.liqHeat) return '—';
    const grade = heatWord(row.liqHeat);
    if (!row.liqHeatVenue) return grade;
    return `${grade} · ${venueName(row.liqHeatVenue)}`;
  };
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
      title: t('derivatives:recorded'),
      dataIndex: 'recorded',
      render: (_value, row) => (
        <Text variant="caption" color="secondary" title={recordedTitle(row.recorded, formatRecordedSpan)}>
          {recordedSummary(
            row.recorded,
            (kind) => t(`detail:chart.recordedKind.${kind}`, { defaultValue: kind }),
            formatRecordedSpan,
          )}
        </Text>
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
      title: t('derivatives:gap'),
      dataIndex: 'gapPct',
      ...sortable('gapPct'),
      render: (_value, row) => (
        <Text variant="caption" title={gapTitle(row)}>
          {definiteText(row.gapDefinite, row.gapPct, (value) => value)}
        </Text>
      ),
    },
    {
      title: t('derivatives:cvdDay'),
      dataIndex: 'cvdDay',
      ...sortable('cvdDay'),
      render: (_value, row) => definiteText(row.cvdDayDefinite, row.cvdDay, signedAmount),
    },
    {
      title: t('derivatives:printSize'),
      dataIndex: 'printChangePct',
      ...sortable('printChange'),
      render: (_value, row) => printLabel(row),
    },
    {
      title: t('derivatives:liqHeat'),
      dataIndex: 'liqHeat',
      ...sortable('liqHeat'),
      render: (_value, row) => heatLabel(row),
    },
    {
      title: t('derivatives:bid'),
      dataIndex: 'bidPct',
      ...sortable('bidPct'),
      render: (_value, row) => percentText(row.bidDefinite, row.bidPct),
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
      scroll={{ x: 1880 }}
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
