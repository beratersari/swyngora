import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  formatRecordPnl,
  situationRecordCloseLine,
  situationRecordKindLabel,
} from '@/components/organisms/PortfolioSituationRecord';
import { reasonHistoryHasCloses, selectedReasonHistory, type ReasonHistoryBucket } from './helpers';
import { CloseList, Groups, List, OpenButton, Row } from './OrderReasonHistory.styles';
import type { OrderReasonHistoryProps } from './OrderReasonHistory.types';

function GroupLine({
  clause,
  kind,
  group,
  bucket,
  openKeys,
  onToggle,
}: {
  clause: string;
  kind: string;
  group: 'still' | 'left';
  bucket: ReasonHistoryBucket | undefined;
  openKeys: string[];
  onToggle: (key: string) => void;
}) {
  const { t, i18n } = useTranslation('portfolio');
  const count = bucket?.count ?? 0;
  const closes = bucket?.closes ?? [];
  const pnl = bucket?.realizedPnL;
  const summary = t(`situationLeave.${group}`, {
    count,
    profitable: bucket?.profitable ?? 0,
    pnl: formatRecordPnl(pnl),
  });
  const key = `${clause}:${kind}:${group}`;
  const open = openKeys.includes(key);
  const panelId = `order-reason-history-${clause}-${kind}-${group}`;
  const summaryText = (
    <Text as="span" variant="caption" color={pnl != null && pnl < 0 ? 'error' : 'secondary'}>
      {summary}
    </Text>
  );
  return (
    <div>
      {closes.length > 0 ? (
        <OpenButton
          type="button"
          aria-expanded={open}
          aria-controls={panelId}
          onClick={() => onToggle(key)}
        >
          {summaryText}
        </OpenButton>
      ) : (
        summaryText
      )}
      {open && closes.length > 0 ? (
        <CloseList id={panelId}>
          {closes.map((close, index) => (
            <li
              key={`${close.exchange ?? ''}:${close.symbol ?? ''}:${close.closedAt ?? ''}:${close.realizedPnL ?? ''}:${index}`}
            >
              <Text
                variant="caption"
                color={close.realizedPnL != null && close.realizedPnL < 0 ? 'error' : 'secondary'}
              >
                {situationRecordCloseLine(close, t, i18n.language)}
              </Text>
            </li>
          ))}
        </CloseList>
      ) : null}
    </div>
  );
}

/** Past closes for the reason clauses still checked on a buy preview. */
export function OrderReasonHistory({ rows, selected }: OrderReasonHistoryProps) {
  const { t } = useTranslation('portfolio');
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const shown = selectedReasonHistory(rows, selected);
  if (shown.length === 0) return null;
  const toggle = (key: string) => {
    setOpenKeys((current) =>
      current.includes(key) ? current.filter((item) => item !== key) : [...current, key],
    );
  };
  return (
    <section aria-labelledby="order-reason-history-title">
      <Text id="order-reason-history-title" variant="caption">
        {t('trade.reasonHistoryTitle')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('trade.reasonHistoryHint')}
      </Text>
      <List>
        {shown.map((row) => {
          const clause = row.clause ?? '';
          const kind = row.kind ?? '';
          return (
            <Row key={`${clause}:${kind}`}>
              <Text variant="body">{situationRecordKindLabel(clause, kind, t)}</Text>
              {reasonHistoryHasCloses(row) ? (
                <Groups>
                  <GroupLine
                    clause={clause}
                    kind={kind}
                    group="still"
                    bucket={row.still}
                    openKeys={openKeys}
                    onToggle={toggle}
                  />
                  <GroupLine
                    clause={clause}
                    kind={kind}
                    group="left"
                    bucket={row.left}
                    openKeys={openKeys}
                    onToggle={toggle}
                  />
                </Groups>
              ) : (
                <Text variant="caption" color="secondary">
                  {t('trade.reasonHistoryNone')}
                </Text>
              )}
            </Row>
          );
        })}
      </List>
    </section>
  );
}
