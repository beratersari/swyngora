import { Button } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import type { ScannerResult } from '@/libs/api';
import { formatDateTime, formatSymbolDisplay, ruleTypeShort } from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import type { SignalsHitsTableProps } from './SignalsHitsTable.types';

export function SignalsHitsTable({ items, loading, onOpen }: SignalsHitsTableProps) {
  const { t } = useTranslation(['signals', 'common']);

  const columns: ColumnsType<ScannerResult> = [
    {
      title: t('signals:symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Text variant="label" mono color="primary">
          {formatSymbolDisplay(row.symbol)}
        </Text>
      ),
    },
    {
      title: t('signals:exchange'),
      dataIndex: 'exchange',
      key: 'exchange',
      render: (ex: string) => <BrandTag variant="exchange">{ex}</BrandTag>,
    },
    {
      title: t('signals:factor'),
      key: 'ruleType',
      render: (_, row) => <BrandTag variant="status">{ruleTypeShort(row.ruleType)}</BrandTag>,
    },
    {
      title: t('signals:interval'),
      dataIndex: 'interval',
      key: 'interval',
      render: (iv: string) => (
        <Text variant="caption" mono>
          {iv}
        </Text>
      ),
    },
    {
      title: t('signals:summary'),
      dataIndex: 'summary',
      key: 'summary',
      ellipsis: true,
      render: (s: string) => <Text variant="body">{s}</Text>,
    },
    {
      title: t('signals:matchedAt'),
      dataIndex: 'matchedAt',
      key: 'matchedAt',
      render: (v: string) => (
        <Text variant="caption" color="secondary">
          {formatDateTime(v)}
        </Text>
      ),
    },
    {
      title: t('signals:actions'),
      key: 'actions',
      render: (_, row) =>
        onOpen ? (
          <Button size="small" type="link" onClick={() => onOpen(row.exchange, row.symbol)}>
            {t('signals:openChart')}
          </Button>
        ) : null,
    },
  ];

  return (
    <DataTableCard>
      <DataTable
        rowKey={(r) => r.id}
        loading={loading}
        dataSource={items}
        columns={columns}
        pagination={{ pageSize: 20 }}
        locale={{ emptyText: <DeskEmpty title={t('signals:hits.empty')} /> }}
        size="small"
        scroll={{ x: 640 }}
      />
    </DataTableCard>
  );
}
