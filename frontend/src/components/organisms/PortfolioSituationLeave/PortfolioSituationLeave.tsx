import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  formatRecordPnl,
  situationRecordCloseLine,
  situationRecordKindLabel,
} from '@/components/organisms/PortfolioSituationRecord';
import { situationLeaveGroupKey } from './helpers';
import { CloseList, Groups, List, OpenButton, Row } from './PortfolioSituationLeave.styles';
import type { PortfolioSituationLeaveProps } from './PortfolioSituationLeave.types';

type LeaveClose = {
  exchange?: string;
  symbol?: string;
  closedAt?: string;
  realizedPnL?: number;
};

type LeaveGroup = {
  count?: number;
  profitable?: number;
  realizedPnL?: number;
  closes?: LeaveClose[];
};

function LeaveGroupLines({
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
  bucket: LeaveGroup | undefined;
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
  const summaryColor = pnl != null && pnl < 0 ? 'error' : 'secondary';
  const key = situationLeaveGroupKey(clause, kind, group);
  const open = openKeys.includes(key);
  const panelId = `situation-leave-${clause}-${kind}-${group}`;
  const summaryText = (
    <Text as="span" variant="caption" color={summaryColor}>
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
                color={
                  close.realizedPnL != null && close.realizedPnL < 0 ? 'error' : 'secondary'
                }
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

export function PortfolioSituationLeave({ record, loading, failed }: PortfolioSituationLeaveProps) {
  const { t } = useTranslation('portfolio');
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  const rows = record?.rows ?? [];
  const toggle = (key: string) => {
    setOpenKeys((current) =>
      current.includes(key) ? current.filter((item) => item !== key) : [...current, key],
    );
  };
  const quiet = !loading && !failed && rows.length === 0;
  return (
    <section aria-labelledby="situation-leave-title">
      <Text id="situation-leave-title" variant="h3">
        {t('situationLeave.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('situationLeave.lead')}
      </Text>
      {loading && rows.length === 0 ? (
        <Text variant="body" color="secondary">
          {t('situationLeave.loading')}
        </Text>
      ) : null}
      {failed && rows.length === 0 ? (
        <Text variant="body" color="secondary">
          {t('situationLeave.failed')}
        </Text>
      ) : null}
      {quiet ? (
        <Text variant="body" color="secondary">
          {t('situationLeave.empty')}
        </Text>
      ) : null}
      {rows.length > 0 ? (
        <List>
          {rows.map((row) => {
            const clause = row.clause ?? '';
            const kind = row.kind ?? '';
            const label = situationRecordKindLabel(clause, kind, t);
            return (
              <Row key={`${clause}:${kind}`}>
                <Text variant="body">{label}</Text>
                <Groups>
                  <LeaveGroupLines
                    clause={clause}
                    kind={kind}
                    group="still"
                    bucket={row.still}
                    openKeys={openKeys}
                    onToggle={toggle}
                  />
                  <LeaveGroupLines
                    clause={clause}
                    kind={kind}
                    group="left"
                    bucket={row.left}
                    openKeys={openKeys}
                    onToggle={toggle}
                  />
                </Groups>
              </Row>
            );
          })}
        </List>
      ) : null}
      {(record?.skipped ?? 0) > 0 ? (
        <Text variant="caption" color="secondary">
          {t('situationLeave.skipped', { count: record?.skipped ?? 0 })}
        </Text>
      ) : null}
      {(record?.unreadable ?? 0) > 0 ? (
        <Text variant="caption" color="secondary">
          {t('situationLeave.unreadable', { count: record?.unreadable ?? 0 })}
        </Text>
      ) : null}
    </section>
  );
}
