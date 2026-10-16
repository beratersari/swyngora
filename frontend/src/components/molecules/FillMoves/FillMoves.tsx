import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { FillMovesProps } from './FillMoves.types';
import { fillMoveLine, fillMoveTone } from './helpers';

export function FillMoves({ moves }: FillMovesProps) {
  const { t } = useTranslation('portfolio');
  if (!moves || moves.length === 0) return null;
  return (
    <div role="group" aria-label={t('moves.title')} title={t('moves.hint')}>
      {moves.map((move, index) => (
        <Text
          key={`${move.window ?? 'window'}-${index}`}
          as="div"
          variant="caption"
          color={fillMoveTone(move)}
        >
          {fillMoveLine(move, t)}
        </Text>
      ))}
    </div>
  );
}
