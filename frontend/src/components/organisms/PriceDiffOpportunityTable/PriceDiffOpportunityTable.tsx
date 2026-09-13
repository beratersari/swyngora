import { Segmented, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatArbPct } from '@/components/organisms/PriceDiffScanTable';
import {
  closeReasonLabel,
  formatOppDuration,
  formatOppTime,
  lastNetLabel,
  oppRoute,
  sparklinePoints,
} from './helpers';
import { Review, Spark } from './PriceDiffOpportunityTable.styles';
import type { PriceDiffOpportunityTableProps } from './PriceDiffOpportunityTable.types';

export function PriceDiffOpportunityTable({
  opportunities,
  isLoading,
  status,
  onStatusChange,
  expandedId,
  onExpand,
  detail,
  detailLoading,
}: PriceDiffOpportunityTableProps) {
  const { t } = useTranslation('arb');
  const ticks = detail && detail.id === expandedId ? detail.ticks : undefined;
  const points = sparklinePoints(ticks);

  return (
    <>
      <Segmented
        value={status}
        options={[
          { value: 'open', label: t('price.statusFilters.open') },
          { value: 'closed', label: t('price.statusFilters.closed') },
          { value: 'all', label: t('price.statusFilters.all') },
        ]}
        onChange={(v) => onStatusChange(v as typeof status)}
      />
      <Table
        size="small"
        rowKey={(r) => r.id ?? `${r.symbol}-${r.buyExchange}-${r.openedAt}`}
        loading={isLoading}
        pagination={false}
        dataSource={opportunities}
        locale={{ emptyText: t('price.noOpps') }}
        expandable={{
          expandedRowKeys: expandedId ? [expandedId] : [],
          onExpand: (open, row) => onExpand(open ? row.id : undefined),
          expandedRowRender: (row) => {
            const review = (detail?.id === row.id ? detail.review : row.review) ?? row.review;
            return (
              <Review>
                <Text variant="bodySm">{review?.summary || t('price.reviewEmpty')}</Text>
                <Text variant="caption" color="secondary">
                  {t('price.opened')}: {formatOppTime(row.openedAt)}
                  {row.review?.peakAt ? ` · ${t('price.peaked')}: ${formatOppTime(row.review.peakAt)}` : ''}
                  {row.fadeStartedAt || row.review?.fadeStartedAt
                    ? ` · ${t('price.faded')}: ${formatOppTime(row.fadeStartedAt ?? row.review?.fadeStartedAt)}`
                    : ''}
                  {row.closedAt ? ` · ${t('price.closed')}: ${formatOppTime(row.closedAt)}` : ''}
                </Text>
                {detailLoading && expandedId === row.id ? (
                  <Text variant="caption" color="secondary">
                    {t('price.pathLoading')}
                  </Text>
                ) : points ? (
                  <Spark viewBox="0 0 160 36" aria-label={t('price.pathAria')}>
                    <polyline fill="none" stroke="currentColor" strokeWidth="1.5" points={points} />
                  </Spark>
                ) : null}
              </Review>
            );
          },
        }}
        columns={[
          { title: t('price.symbol'), dataIndex: 'symbol' },
          { title: t('price.route'), render: (_, r) => oppRoute(r) },
          {
            title: t('price.duration'),
            render: (_, r) => formatOppDuration(r.review?.openDurationSec),
          },
          {
            title: t('price.openPct'),
            render: (_, r) => formatArbPct(r.review?.openNetDiffPct ?? r.openNetDiffPct),
          },
          {
            title: t('price.peakPct'),
            render: (_, r) => formatArbPct(r.review?.peakNetDiffPct ?? r.peakNetDiffPct),
          },
          { title: t('price.netPct'), render: (_, r) => lastNetLabel(r) },
          {
            title: t('price.whyClosed'),
            render: (_, r) =>
              r.status === 'closed' ? closeReasonLabel(t, r.closeReason ?? r.review?.closeReason) : '—',
          },
        ]}
      />
    </>
  );
}
