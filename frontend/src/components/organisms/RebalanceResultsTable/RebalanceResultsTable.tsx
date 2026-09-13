import { Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  formatRebalanceMoney,
  formatRebalancePct,
  formatRebalanceWhen,
  legStatusKey,
  rebalanceStatusKey,
} from './helpers';
import { Detail } from './RebalanceResultsTable.styles';
import type { RebalanceResultsTableProps } from './RebalanceResultsTable.types';

export function RebalanceResultsTable({
  results,
  isLoading,
  expandedId,
  onExpand,
  detail,
  detailLoading,
}: RebalanceResultsTableProps) {
  const { t } = useTranslation('portfolio');
  const open = detail && detail.id === expandedId ? detail : undefined;

  return (
    <Table
      size="small"
      rowKey={(r) => r.id ?? `${r.basketId}-${r.createdAt}`}
      loading={isLoading}
      pagination={false}
      dataSource={results}
      locale={{ emptyText: t('rebalance.empty') }}
      expandable={{
        expandedRowKeys: expandedId ? [expandedId] : [],
        onExpand: (next, row) => onExpand(next ? row.id : undefined),
        expandedRowRender: (row) => {
          const body = open?.id === row.id ? open : row;
          return (
            <Detail>
              <Text variant="bodySm">{body.summary || t('rebalance.noSummary')}</Text>
              {detailLoading && expandedId === row.id ? (
                <Text variant="caption" color="secondary">
                  {t('rebalance.loadingDetail')}
                </Text>
              ) : null}
              <Table
                size="small"
                pagination={false}
                rowKey={(l, i) => `${l.symbol}-${l.side}-${i}`}
                dataSource={body.legs ?? []}
                locale={{ emptyText: t('rebalance.noLegs') }}
                columns={[
                  { title: t('rebalance.side'), dataIndex: 'side' },
                  { title: t('rebalance.asset'), dataIndex: 'asset' },
                  {
                    title: t('rebalance.legStatus'),
                    render: (_, l) => t(legStatusKey(l.status)),
                  },
                  {
                    title: t('rebalance.planned'),
                    render: (_, l) => formatRebalanceMoney(l.plannedNotional, body.currency),
                  },
                  {
                    title: t('rebalance.filled'),
                    render: (_, l) =>
                      l.status === 'filled' ? formatRebalanceMoney(l.fillNotional, body.currency) : '—',
                  },
                  {
                    title: t('rebalance.error'),
                    render: (_, l) => l.error || '—',
                  },
                ]}
              />
              <Table
                size="small"
                pagination={false}
                rowKey={(ln) => ln.asset ?? 'cash'}
                dataSource={body.lines ?? []}
                locale={{ emptyText: t('rebalance.noLines') }}
                columns={[
                  { title: t('rebalance.asset'), dataIndex: 'asset' },
                  { title: t('rebalance.target'), render: (_, ln) => formatRebalancePct(ln.targetPct) },
                  { title: t('rebalance.before'), render: (_, ln) => formatRebalancePct(ln.beforePct) },
                  { title: t('rebalance.after'), render: (_, ln) => formatRebalancePct(ln.afterPct) },
                  {
                    title: t('rebalance.leftover'),
                    render: (_, ln) => formatRebalanceMoney(ln.afterDelta, body.currency),
                  },
                ]}
              />
            </Detail>
          );
        },
      }}
      columns={[
        { title: t('rebalance.when'), render: (_, r) => formatRebalanceWhen(r.createdAt) },
        { title: t('rebalance.basket'), dataIndex: 'basketName' },
        {
          title: t('rebalance.status'),
          render: (_, r) => t(rebalanceStatusKey(r.status)),
        },
        {
          title: t('rebalance.fills'),
          render: (_, r) => `${r.filledLegs ?? 0}/${r.plannedLegs ?? 0}`,
        },
        {
          title: t('rebalance.leftoverPct'),
          render: (_, r) => formatRebalancePct(r.remainingAbsDriftPct),
        },
      ]}
    />
  );
}
