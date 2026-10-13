import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { situationChangeLines, situationLines } from '@/components/molecules/CoinSituation';
import { openLotCaption, type OpenLotReadView } from './helpers';
import { Lot, Stack } from './PositionSituation.styles';
import type { PositionSituationProps } from './PositionSituation.types';

export function PositionSituation({
  entry,
  now,
  nowLoading,
  comparison,
  lots,
}: PositionSituationProps) {
  const { t, i18n } = useTranslation('detail');
  const openLots = (lots ?? []).filter((lot) => (lot.quantity ?? 0) > 0);
  const entered = situationLines(entry, t);
  const current = situationLines(now, t, { since: true, locale: i18n.language });
  const changed = situationChangeLines(entry, now, comparison, 'live', t);
  if (openLots.length === 0 && entered.length === 0 && current.length === 0 && !nowLoading)
    return null;
  return (
    <Stack role="group" aria-label={t('situation.compare')}>
      {openLots.length > 0
        ? openLots.map((lot, index) => (
            <LotRead
              key={lot.id || lot.sourceTradeId || `${lot.openedAt ?? ''}-${index}`}
              lot={lot}
              newest={lot.newest === true || (lot.newest == null && index === 0)}
              several={openLots.length > 1}
              now={now}
              locale={i18n.language}
            />
          ))
        : null}
      {openLots.length === 0 && entered.length > 0 ? (
        <>
          <Text variant="caption" color="secondary">
            {t('situation.entered')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('situation.atFill')}
          </Text>
          {entered.map((line) => (
            <Text key={`entered-${line}`} variant="caption">
              {line}
            </Text>
          ))}
        </>
      ) : null}
      {openLots.length === 0
        ? changed.map((line) => (
            <Text key={`left-${line}`} variant="caption">
              {line}
            </Text>
          ))
        : null}
      {nowLoading && current.length === 0 ? (
        <Text variant="caption" color="secondary">
          {t('situation.nowLoading')}
        </Text>
      ) : null}
      {current.length > 0 ? (
        <>
          <Text variant="caption" color="secondary">
            {t('situation.now')}
          </Text>
          {current.map((line) => (
            <Text key={`now-${line}`} variant="caption">
              {line}
            </Text>
          ))}
        </>
      ) : null}
    </Stack>
  );
}

function LotRead({
  lot,
  newest,
  several,
  now,
  locale,
}: {
  lot: OpenLotReadView;
  newest: boolean;
  several: boolean;
  now?: PositionSituationProps['now'];
  locale: string;
}) {
  const { t } = useTranslation('detail');
  const lines = situationLines(lot.situation, t);
  const changed = situationChangeLines(lot.situation, now, lot.comparison, 'live', t);
  return (
    <Lot>
      <Text variant="caption" color="secondary">
        {newest ? t('situation.entered') : t('situation.earlier')}
      </Text>
      <Text variant="caption" color="secondary">
        {openLotCaption(lot.quantity, lot.openedAt, locale, t)}
      </Text>
      {newest && several ? (
        <Text variant="caption" color="secondary">
          {t('situation.alertFollows')}
        </Text>
      ) : null}
      {lines.length > 0 ? (
        <Text variant="caption" color="secondary">
          {t('situation.atFill')}
        </Text>
      ) : (
        <Text variant="caption" color="secondary">
          {t('situation.lotBlank')}
        </Text>
      )}
      {lines.map((line, index) => (
        <Text key={`${lot.id ?? 'lot'}-c-${index}`} variant="caption">
          {line}
        </Text>
      ))}
      {changed.map((line, index) => (
        <Text key={`${lot.id ?? 'lot'}-l-${index}`} variant="caption">
          {line}
        </Text>
      ))}
    </Lot>
  );
}
