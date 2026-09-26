import { Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { FundingArbHistoryTableProps } from './FundingArbHistoryTable.types';

export function FundingArbHistoryTable({ history, isLoading }: FundingArbHistoryTableProps) {
  const { t } = useTranslation('arb');
  const rows = history?.runs ?? [];
  return (
    <>
      {history?.summary ? (
        <Text variant="caption" color="secondary">
          {history.summary}
        </Text>
      ) : null}
      <Table
        size="small"
        rowKey={(r) => `${r.startedAt}-${r.longExchange}-${r.shortExchange}`}
        loading={isLoading}
        pagination={false}
        dataSource={rows}
        locale={{ emptyText: t('funding.historyEmpty') }}
        columns={[
          {
            title: t('funding.started'),
            render: (_, r) => <Text variant="bodySm">{r.startedAt ?? '—'}</Text>,
          },
          {
            title: t('funding.ended'),
            render: (_, r) => <Text variant="bodySm">{r.endedAt ?? '—'}</Text>,
          },
          {
            title: t('funding.long'),
            render: (_, r) => <Text variant="bodySm">{r.longExchange ?? '—'}</Text>,
          },
          {
            title: t('funding.short'),
            render: (_, r) => <Text variant="bodySm">{r.shortExchange ?? '—'}</Text>,
          },
          {
            title: t('funding.payments'),
            render: (_, r) => <Text variant="numeric">{r.paymentCount ?? '—'}</Text>,
          },
          {
            title: t('funding.net'),
            render: (_, r) => <Text variant="numeric">{r.netAfterFees ?? '—'}</Text>,
          },
          {
            title: t('funding.read'),
            render: (_, r) => <Text variant="caption">{r.summary ?? '—'}</Text>,
          },
        ]}
      />
    </>
  );
}
