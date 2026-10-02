import { Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { MarketUnusualScanHit } from '@/libs/api';
import { unusualHitKey, unusualSpanKey, unusualTapePath, unusualValue } from './helpers';
import type { UnusualScanBoardProps } from './UnusualScanBoard.types';

export function UnusualScanBoard({ hits, loading }: UnusualScanBoardProps) {
  const { t } = useTranslation(['unusual', 'common']);
  const columns: ColumnsType<MarketUnusualScanHit> = [
    {
      title: t('coin'),
      dataIndex: 'base',
      fixed: 'left',
      render: (_value, row) => (
        <Link to={unusualTapePath(row.exchange, row.symbol ?? '')}>{row.base || row.symbol}</Link>
      ),
    },
    {
      title: t('venue'),
      dataIndex: 'exchange',
      render: (exchange: string) => t(`common:exchanges.${exchange}`, { defaultValue: exchange }),
    },
    {
      title: t('metric'),
      dataIndex: 'metric',
      render: (metric: string) => t(`metrics.${metric}`, { defaultValue: metric }),
    },
    {
      title: t('band'),
      dataIndex: 'band',
      defaultSortOrder: 'descend',
      sorter: (a, b) => (a.distance ?? 0) - (b.distance ?? 0),
      render: (band: string) => t(`bands.${band}`, { defaultValue: band }),
    },
    {
      title: t('percentile'),
      dataIndex: 'percentile',
      sorter: (a, b) => (a.percentile ?? 0) - (b.percentile ?? 0),
      render: (value: number | undefined) => (value == null ? '—' : String(Math.round(value))),
    },
    {
      title: t('value'),
      key: 'value',
      render: (_value, row) => unusualValue(row),
    },
    {
      title: t('samples'),
      key: 'samples',
      render: (_value, row) => {
        const span = unusualSpanKey(row.spanHours);
        return t('sampleCount', {
          count: row.samples ?? 0,
          span: t(span.key, span.count == null ? undefined : { count: span.count }),
        });
      },
    },
  ];

  return (
    <Table<MarketUnusualScanHit>
      size="small"
      rowKey={unusualHitKey}
      loading={loading}
      columns={columns}
      dataSource={hits}
      pagination={hits.length > 20 ? { pageSize: 20, showSizeChanger: false } : false}
      scroll={{ x: 880 }}
      locale={{ emptyText: t('empty') }}
    />
  );
}
