import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { FillPocketProps } from './FillPocket.types';
import { fillPocketSentences } from './helpers';

export function FillPocket({ mark }: FillPocketProps) {
  const { t } = useTranslation('detail');
  const lines = fillPocketSentences(mark, t);
  if (lines.length === 0) return null;
  return (
    <div role="group" aria-label={t('fillPocket.title')} data-testid="fill-pocket">
      {lines.map((line) => (
        <Text key={line} as="div" variant="caption" color="secondary">
          {line}
        </Text>
      ))}
    </div>
  );
}
