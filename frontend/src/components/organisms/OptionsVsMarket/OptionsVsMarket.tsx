import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { vsMarketFindings } from './helpers';
import { Card, Stack, Topic } from './OptionsVsMarket.styles';
import type { OptionsVsMarketProps } from './OptionsVsMarket.types';

export function OptionsVsMarket({ data, isLoading }: OptionsVsMarketProps) {
  const { t } = useTranslation('options');
  const findings = vsMarketFindings(data?.vsMarket?.findings);

  if (isLoading && !data) {
    return <Skeleton height={96} />;
  }
  if (findings.length === 0) {
    return null;
  }

  return (
    <Stack>
      {findings.map((row) => (
        <Card key={row.kind ?? row.title} $notable={row.severity === 'notable'}>
          <Text variant="caption" color="secondary">
            <Topic>{t(`vs.topic.${row.topic ?? 'vol'}` as 'vs.topic.vol')}</Topic>
            {row.severity === 'notable' ? t('vs.notable') : t('vs.watch')}
          </Text>
          <Text variant="bodySm" color="primary">
            {row.title}
          </Text>
          {row.detail ? (
            <Text variant="caption" color="secondary">
              {row.detail}
            </Text>
          ) : null}
        </Card>
      ))}
    </Stack>
  );
}
