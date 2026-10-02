import { Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { DerivativesCoin, DerivativesVenue } from '@/libs/api';
import { coinChartPath, definiteText, fundingText, plainAmount, priceText, signedAmount } from './helpers';
import type { DerivativesBoardProps } from './DerivativesBoard.types';

export function DerivativesBoard({ exchange, coins, loading }: DerivativesBoardProps) {
  const { t } = useTranslation(['derivatives', 'common']);
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
  const columns: ColumnsType<DerivativesCoin> = [
    {
      title: t('derivatives:coin'),
      dataIndex: 'symbol',
      fixed: 'left',
      render: (_value, row) => (
        <Link to={coinChartPath(exchange, row.symbol ?? '')}>{row.base || row.symbol}</Link>
      ),
    },
    {
      title: t('derivatives:price'),
      dataIndex: 'price',
      render: (_value, row) => priceText(row),
    },
    {
      title: t('derivatives:funding'),
      dataIndex: 'fundingRatePct',
      render: (_value, row) => fundingText(row),
    },
    {
      title: t('derivatives:openInterest'),
      dataIndex: 'openInterest',
      render: (_value, row) => definiteText(row.openInterestDefinite, row.openInterest, plainAmount),
    },
    {
      title: t('derivatives:change1h'),
      dataIndex: 'change1h',
      render: (_value, row) => definiteText(row.change1hDefinite, row.change1h, signedAmount),
    },
    {
      title: t('derivatives:change24h'),
      dataIndex: 'change24h',
      render: (_value, row) => definiteText(row.change24hDefinite, row.change24h, signedAmount),
    },
    {
      title: t('derivatives:liqLong'),
      dataIndex: 'liqLong',
      render: (_value, row) => definiteText(row.liqDefinite, row.liqLong, plainAmount),
    },
    {
      title: t('derivatives:liqShort'),
      dataIndex: 'liqShort',
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

  return (
    <Table<DerivativesCoin>
      rowKey={(row) => row.symbol ?? ''}
      columns={columns}
      dataSource={coins}
      loading={loading}
      pagination={false}
      scroll={{ x: 960 }}
      locale={{ emptyText: t('derivatives:empty') }}
    />
  );
}
