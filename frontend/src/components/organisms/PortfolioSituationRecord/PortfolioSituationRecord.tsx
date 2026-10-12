import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatRecordPnl, situationRecordKindLabel } from './helpers';
import { List, Row } from './PortfolioSituationRecord.styles';
import type { PortfolioSituationRecordProps } from './PortfolioSituationRecord.types';

export function PortfolioSituationRecord({ record, loading, failed }: PortfolioSituationRecordProps) {
  const { t } = useTranslation('portfolio');
  const rows = record?.rows ?? [];
  return (
    <section aria-labelledby="situation-record-title">
      <Text id="situation-record-title" variant="h3">
        {t('situationRecord.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('situationRecord.lead')}
      </Text>
      {loading && rows.length === 0 ? (
        <Text variant="body" color="secondary">
          {t('situationRecord.loading')}
        </Text>
      ) : null}
      {failed && rows.length === 0 ? (
        <Text variant="body" color="secondary">
          {t('situationRecord.failed')}
        </Text>
      ) : null}
      {!loading && !failed && rows.length === 0 ? (
        <Text variant="body" color="secondary">
          {t('situationRecord.empty')}
        </Text>
      ) : null}
      {rows.length > 0 ? (
        <List>
          {rows.map((row) => {
            const clause = row.clause ?? '';
            const kind = row.kind ?? '';
            const count = row.count ?? 0;
            return (
              <Row key={`${clause}:${kind}`}>
                <Text variant="body">{situationRecordKindLabel(clause, kind, t)}</Text>
                <Text variant="caption" color={row.realizedPnL != null && row.realizedPnL < 0 ? 'error' : 'secondary'}>
                  {t('situationRecord.row', {
                    count,
                    profitable: row.profitable ?? 0,
                    pnl: formatRecordPnl(row.realizedPnL),
                  })}
                </Text>
              </Row>
            );
          })}
        </List>
      ) : null}
      {(record?.skipped ?? 0) > 0 ? (
        <Text variant="caption" color="secondary">
          {t('situationRecord.skipped', { count: record?.skipped ?? 0 })}
        </Text>
      ) : null}
    </section>
  );
}
