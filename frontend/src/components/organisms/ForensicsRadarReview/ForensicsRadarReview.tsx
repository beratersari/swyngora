import { Skeleton, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { ForensicsRadarCase } from '@/libs/types/forensics.types';
import { formatForensicsTime } from '@/libs/utils/forensics';
import { radarAfterCell } from './helpers';
import { Stack, TableWrap } from './ForensicsRadarReview.styles';
import type { ForensicsRadarReviewProps } from './ForensicsRadarReview.types';

export function ForensicsRadarReview({ review, isLoading, onOpenCase }: ForensicsRadarReviewProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }
  const cases = review?.cases ?? [];
  return (
    <Stack>
      {review?.summary ? <Text variant="bodySm">{review.summary}</Text> : null}
      {review?.note ? (
        <Text variant="caption" color="secondary">
          {review.note}
        </Text>
      ) : null}
      {review?.horizons && review.horizons.length > 0 ? (
        <TableWrap>
          <Table
            size="small"
            pagination={false}
            rowKey={(row) => row.horizon ?? ''}
            dataSource={review.horizons}
            columns={[
              { title: t('horizon'), dataIndex: 'horizon', key: 'horizon' },
              { title: t('sample'), dataIndex: 'sample', key: 'sample' },
              {
                title: t('review.useful'),
                key: 'useful',
                render: (_: unknown, row) =>
                  row.leanRight != null || row.leanWrong != null
                    ? `${row.leanRight ?? 0}/${(row.leanRight ?? 0) + (row.leanWrong ?? 0)} ${row.usefulPct ?? ''}`.trim()
                    : '—',
              },
              { title: t('median'), dataIndex: 'medianPct', key: 'medianPct' },
              { title: t('up'), dataIndex: 'up', key: 'up' },
              { title: t('down'), dataIndex: 'down', key: 'down' },
            ]}
          />
        </TableWrap>
      ) : null}
      {cases.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('review.empty')}
        </Text>
      ) : (
        <TableWrap>
          <Table<ForensicsRadarCase>
            size="small"
            pagination={false}
            rowKey={(row) => row.id ?? `${row.symbol}:${row.foundAt}`}
            dataSource={cases}
            onRow={(row) => ({
              onClick: () => {
                if (row.symbol && row.foundAt) onOpenCase?.(row.symbol, row.foundAt);
              },
              style: onOpenCase ? { cursor: 'pointer' } : undefined,
            })}
            columns={[
              {
                title: t('at'),
                key: 'foundAt',
                render: (_: unknown, row) => formatForensicsTime(row.foundAt),
              },
              { title: t('symbol'), dataIndex: 'symbol', key: 'symbol' },
              { title: t('similarity'), dataIndex: 'similarity', key: 'similarity' },
              {
                title: t('scan.lean'),
                dataIndex: 'lean',
                key: 'lean',
                render: (lean: string) => t(`scan.leans.${lean}`, { defaultValue: lean }),
              },
              {
                title: t('review.after15m'),
                key: 'a15',
                render: (_: unknown, row) => radarAfterCell(row, '15m'),
              },
              {
                title: t('review.after1h'),
                key: 'a1h',
                render: (_: unknown, row) => radarAfterCell(row, '1h'),
              },
              {
                title: t('review.after4h'),
                key: 'a4h',
                render: (_: unknown, row) => radarAfterCell(row, '4h'),
              },
              { title: t('scan.whySimilar'), dataIndex: 'whySimilar', key: 'whySimilar' },
            ]}
          />
        </TableWrap>
      )}
    </Stack>
  );
}
