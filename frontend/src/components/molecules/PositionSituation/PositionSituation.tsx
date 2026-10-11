import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { situationLines } from '@/components/molecules/CoinSituation';
import { Stack } from './PositionSituation.styles';
import type { PositionSituationProps } from './PositionSituation.types';

export function PositionSituation({ lead, entry, now, nowLoading }: PositionSituationProps) {
  const { t } = useTranslation('detail');
  const entered = situationLines(entry, t);
  const current = situationLines(now, t);
  if (entered.length === 0 && current.length === 0 && !nowLoading) return null;
  return (
    <Stack role="group" aria-label={lead || t('situation.compare')}>
      {lead ? (
        <Text variant="caption" color="secondary">
          {lead}
        </Text>
      ) : null}
      {entered.length > 0 ? (
        <>
          <Text variant="caption" color="secondary">
            {t('situation.entered')}
          </Text>
          {entered.map((line) => (
            <Text key={`entered-${line}`} variant="caption">
              {line}
            </Text>
          ))}
        </>
      ) : null}
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
