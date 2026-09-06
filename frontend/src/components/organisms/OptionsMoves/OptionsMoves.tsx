import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { unusualMoves } from './helpers';
import { Card, Stack, Topic } from './OptionsMoves.styles';
import type { OptionsMovesProps } from './OptionsMoves.types';

export function OptionsMoves({ data, isLoading }: OptionsMovesProps) {
  const { t } = useTranslation('options');
  const findings = unusualMoves(data?.moves?.findings);

  if (isLoading && !data) {
    return <Skeleton height={96} />;
  }
  if (findings.length === 0) {
    return null;
  }

  return (
    <Stack>
      {findings.map((row) => (
        <Card key={`${row.kind}-${row.window}`} $notable={row.severity === 'notable'}>
          <Text variant="caption" color="secondary">
            <Topic>{t(`moves.topic.${row.topic ?? 'oi'}` as 'moves.topic.oi')}</Topic>
            {row.window ? t(`moves.window.${row.window}` as 'moves.window.1h') : null}
            {' · '}
            {row.severity === 'notable' ? t('moves.notable') : t('moves.watch')}
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
