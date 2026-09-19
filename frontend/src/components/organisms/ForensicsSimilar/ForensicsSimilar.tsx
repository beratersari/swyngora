import { Skeleton, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { ForensicsSimilarHit } from '@/libs/types/forensics.types';
import { formatForensicsTime } from '@/libs/utils/forensics';
import { orderedSimilarFields, similarFieldScore, similarMoveAt } from './helpers';
import { FieldScores, Stack, TableWrap, WhyCell } from './ForensicsSimilar.styles';
import type { ForensicsSimilarProps } from './ForensicsSimilar.types';

export function ForensicsSimilar({ similar, similarError, isLoading, onOpenMove }: ForensicsSimilarProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 4 }} />;
  }
  return (
    <Stack>
      <Text variant="bodySm">{t('similarTitle')}</Text>
      {similarError ? (
        <Text variant="bodySm" color="secondary">
          {t('similarError')} {similarError}
        </Text>
      ) : null}
      {similar?.summary ? (
        <Text variant="caption" color="secondary">
          {similar.summary}
        </Text>
      ) : null}
      {similar?.afterHorizons && similar.afterHorizons.length > 0 ? (
        <TableWrap>
          <Text variant="caption" color="secondary">
            {t('horizons')}
          </Text>
          <Table
            size="small"
            pagination={false}
            rowKey={(row) => row.horizon ?? ''}
            dataSource={similar.afterHorizons}
            columns={[
              { title: t('horizon'), dataIndex: 'horizon', key: 'horizon' },
              { title: t('up'), dataIndex: 'up', key: 'up' },
              { title: t('down'), dataIndex: 'down', key: 'down' },
              { title: t('median'), dataIndex: 'medianPct', key: 'medianPct' },
              { title: t('sample'), dataIndex: 'sample', key: 'sample' },
            ]}
          />
        </TableWrap>
      ) : null}
      {!similar?.matches?.length && !similarError ? (
        <Text variant="caption" color="secondary">
          {t('similarEmpty')}
        </Text>
      ) : null}
      {similar?.matches && similar.matches.length > 0 ? (
        <TableWrap>
          {onOpenMove ? (
            <Text variant="caption" color="secondary">
              {t('similarOpenHint')}
            </Text>
          ) : null}
          <Table<ForensicsSimilarHit>
            size="small"
            pagination={false}
            rowKey={(row) => `${row.at}-${row.until}`}
            dataSource={similar.matches}
            onRow={(row) => ({
              onClick: () => {
                const at = similarMoveAt(row);
                if (at) onOpenMove?.(at);
              },
              style: onOpenMove ? { cursor: 'pointer' } : undefined,
            })}
            columns={[
              {
                title: t('at'),
                dataIndex: 'at',
                key: 'at',
                render: (v: string) => formatForensicsTime(v),
              },
              { title: t('path'), dataIndex: 'direction', key: 'direction' },
              { title: t('similarity'), dataIndex: 'similarity', key: 'similarity' },
              { title: t('coverage'), dataIndex: 'coverage', key: 'coverage' },
              {
                title: t('scan.whySimilar'),
                key: 'whySimilar',
                render: (_: unknown, row: ForensicsSimilarHit) => {
                  const fields = orderedSimilarFields(row.compared);
                  return (
                    <WhyCell>
                      {row.whySimilar ? <Text variant="bodySm">{row.whySimilar}</Text> : null}
                      {fields.length > 0 ? (
                        <FieldScores>
                          {fields.map((f) => (
                            <Text key={f.name ?? 'field'} variant="caption" color="secondary">
                              {t(`scan.fields.${f.name}`, { defaultValue: f.name })} {similarFieldScore(f)}
                            </Text>
                          ))}
                        </FieldScores>
                      ) : null}
                    </WhyCell>
                  );
                },
              },
              { title: t('afterReturn'), dataIndex: 'afterReturnPct', key: 'afterReturnPct' },
            ]}
          />
        </TableWrap>
      ) : null}
    </Stack>
  );
}
