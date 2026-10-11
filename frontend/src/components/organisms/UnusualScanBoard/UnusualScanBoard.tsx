import { Button, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { OutlierSituation } from '@/components/molecules/OutlierSituation';
import type { MarketUnusualScanHit } from '@/libs/api';
import { unusualHitKey, unusualSpanKey, unusualTapePath, unusualValue } from './helpers';
import type { UnusualScanBoardProps } from './UnusualScanBoard.types';

export function UnusualScanBoard({ hits, loading }: UnusualScanBoardProps) {
  const { t } = useTranslation(['unusual', 'common']);
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const toggleKey = (key: string) => {
    setOpenKeys((current) => (current.includes(key) ? current.filter((item) => item !== key) : [...current, key]));
  };
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
      render: (band: string, row) => {
        const label = t(`bands.${band}`, { defaultValue: band });
        if (!row.symbol) return label;
        const key = unusualHitKey(row);
        return (
          <Button
            type="link"
            size="small"
            aria-expanded={openKeys.includes(key)}
            style={{ height: 'auto', padding: 0, whiteSpace: 'normal', textAlign: 'left' }}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              toggleKey(key);
            }}
          >
            {label}
          </Button>
        );
      },
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
      expandable={{
        showExpandColumn: false,
        expandedRowKeys: openKeys,
        rowExpandable: (row) => Boolean(row.symbol),
        expandedRowRender: (row) => <OutlierSituation symbol={row.symbol ?? ''} />,
      }}
    />
  );
}
