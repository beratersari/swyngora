import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { situationLines } from '@/components/molecules/CoinSituation';
import { Card, Header } from './CoinCandleSituation.styles';
import type { CoinCandleSituationProps } from './CoinCandleSituation.types';

export function CoinCandleSituation({
  timeLabel,
  report,
  isLoading,
  error,
}: CoinCandleSituationProps) {
  const { t, i18n } = useTranslation('detail');
  if (!timeLabel) return null;
  const lines = situationLines(report, t, { since: true, locale: i18n.language });
  return (
    <Card aria-label={t('situation.candle.title')} data-testid="chart-candle-situation">
      <Header>
        <Text variant="label">{t('situation.candle.title')}</Text>
        <Text variant="caption" color="secondary">
          {t('situation.candle.at', { time: timeLabel })}
        </Text>
      </Header>
      <Text variant="caption" color="secondary">
        {t('situation.candle.lead')}
      </Text>
      {isLoading ? (
        <Text variant="caption" color="secondary" data-testid="chart-candle-situation-loading">
          {t('situation.candle.loading')}
        </Text>
      ) : error ? (
        <Text variant="bodySm" data-testid="chart-candle-situation-error">
          {error}
        </Text>
      ) : (
        lines.map((line, index) => (
          <Text key={`${line}-${index}`} variant="bodySm">
            {line}
          </Text>
        ))
      )}
    </Card>
  );
}
