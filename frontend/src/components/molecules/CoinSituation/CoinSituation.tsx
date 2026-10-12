import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Card } from './CoinSituation.styles';
import type { CoinSituationProps } from './CoinSituation.types';
import { situationLines } from './helpers';

export function CoinSituation({ report, isLoading, error }: CoinSituationProps) {
  const { t, i18n } = useTranslation('detail');
  if (error) {
    return (
      <Text variant="caption" color="secondary">
        {error}
      </Text>
    );
  }
  if (isLoading && !report) {
    return (
      <Text variant="caption" color="secondary">
        {t('situation.loading')}
      </Text>
    );
  }
  const lines = situationLines(report, t, { since: true, locale: i18n.language });
  if (lines.length === 0) return null;
  return (
    <Card aria-label={t('situation.title')}>
      <Text variant="label">{t('situation.title')}</Text>
      {lines.map((line) => (
        <Text key={line} variant="body">
          {line}
        </Text>
      ))}
    </Card>
  );
}
