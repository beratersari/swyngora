import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  formatRecordPnl,
  situationRecordCloseLine,
  situationRecordKindLabel,
  situationRecordRowKey,
} from './helpers';
import { CloseList, List, OpenButton, Row } from './PortfolioSituationRecord.styles';
import type { PortfolioSituationRecordProps } from './PortfolioSituationRecord.types';

export function PortfolioSituationRecord({
  record,
  loading,
  failed,
}: PortfolioSituationRecordProps) {
  const { t, i18n } = useTranslation('portfolio');
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const rows = record?.rows ?? [];
  const toggle = (key: string) => {
    setOpenKeys((current) =>
      current.includes(key) ? current.filter((item) => item !== key) : [...current, key],
    );
  };
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
            const key = situationRecordRowKey(clause, kind);
            const label = situationRecordKindLabel(clause, kind, t);
            const summary = t('situationRecord.row', {
              count,
              profitable: row.profitable ?? 0,
              pnl: formatRecordPnl(row.realizedPnL),
            });
            const summaryColor =
              row.realizedPnL != null && row.realizedPnL < 0 ? 'error' : 'secondary';
            const closes = row.closes ?? [];
            const open = openKeys.includes(key);
            const panelId = `situation-record-${clause}-${kind}`;
            return (
              <Row key={key}>
                {closes.length > 0 ? (
                  <OpenButton
                    type="button"
                    aria-expanded={open}
                    aria-controls={panelId}
                    onClick={() => toggle(key)}
                  >
                    <Text as="span" variant="body">
                      {label}
                    </Text>
                    <Text as="span" variant="caption" color={summaryColor}>
                      {summary}
                    </Text>
                  </OpenButton>
                ) : (
                  <>
                    <Text variant="body">{label}</Text>
                    <Text variant="caption" color={summaryColor}>
                      {summary}
                    </Text>
                  </>
                )}
                {open && closes.length > 0 ? (
                  <CloseList id={panelId}>
                    {closes.map((close, index) => (
                      <li
                        key={`${close.exchange ?? ''}:${close.symbol ?? ''}:${close.closedAt ?? ''}:${close.realizedPnL ?? ''}:${index}`}
                      >
                        <Text
                          variant="caption"
                          color={
                            close.realizedPnL != null && close.realizedPnL < 0
                              ? 'error'
                              : 'secondary'
                          }
                        >
                          {situationRecordCloseLine(close, t, i18n.language)}
                        </Text>
                      </li>
                    ))}
                  </CloseList>
                ) : null}
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
