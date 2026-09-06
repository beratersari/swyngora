import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { OPTIONS_STAT_KEYS } from './constants';
import { optionsStatValues } from './helpers';
import { Card, Grid, Label } from './OptionsStatCards.styles';
import type { OptionsStatCardsProps } from './OptionsStatCards.types';

export function OptionsStatCards({ data, isLoading }: OptionsStatCardsProps) {
  const { t } = useTranslation('options');
  const asset = data?.asset ?? 'BTC';
  const vals = optionsStatValues(data, asset);

  if (isLoading && !data) {
    return (
      <Grid>
        {OPTIONS_STAT_KEYS.map((key) => (
          <Skeleton key={key} height={108} />
        ))}
      </Grid>
    );
  }

  return (
    <Grid>
      <Card>
        <Label>
          <Text variant="caption" color="secondary">
            {t('stats.oi')}
          </Text>
        </Label>
        <Text variant="h3" color="primary">
          {vals.oiUsd}
        </Text>
        <Text variant="caption" color="secondary">
          {vals.oiChange24h ? t('stats.change24h', { value: vals.oiChange24h }) : vals.oiCoin}
        </Text>
      </Card>
      <Card>
        <Label>
          <Text variant="caption" color="secondary">
            {t('stats.volume')}
          </Text>
        </Label>
        <Text variant="h3" color="primary">
          {vals.volUsd}
        </Text>
        <Text variant="caption" color="secondary">
          {vals.volCoin}
        </Text>
      </Card>
      <Card>
        <Label>
          <Text variant="caption" color="secondary">
            {t('stats.pcr')}
          </Text>
        </Label>
        <Text variant="h3" color="primary">
          {vals.pcr}
        </Text>
        <Text variant="caption" color="secondary">
          {vals.pcrChange24h
            ? t('stats.change24h', { value: vals.pcrChange24h })
            : vals.lean
              ? t(`lean.${vals.lean}` as 'lean.balanced')
              : t('stats.pcrHint')}
        </Text>
      </Card>
      <Card>
        <Label>
          <Text variant="caption" color="secondary">
            {t('stats.dvol')}
          </Text>
        </Label>
        <Text variant="h3" color="primary">
          {vals.dvol}
        </Text>
        <Text variant="caption" color="secondary">
          {vals.dvolChange ? t('stats.dvolChange', { value: vals.dvolChange }) : t('stats.ivHint')}
        </Text>
      </Card>
      <Card>
        <Label>
          <Text variant="caption" color="secondary">
            {t('stats.atm')}
          </Text>
        </Label>
        <Text variant="h3" color="primary">
          {vals.atm}
        </Text>
        <Text variant="caption" color="secondary">
          {vals.atmDte != null ? t('stats.atmDte', { days: vals.atmDte }) : t('stats.ivHint')}
        </Text>
      </Card>
    </Grid>
  );
}
