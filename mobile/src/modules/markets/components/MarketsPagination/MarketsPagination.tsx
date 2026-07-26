import { View } from 'react-native';
import { Button } from '@/components/atoms/Button';
import { Text } from '@/components/atoms/Text';
import type { MarketsPaginationProps } from './MarketsPagination.types';
import { styles } from './MarketsPagination.styles';

export function MarketsPagination({
  offset,
  limit,
  total,
  canPrev,
  canNext,
  onPrev,
  onNext,
}: MarketsPaginationProps) {
  const start = total === 0 ? 0 : offset + 1;
  const end = Math.min(offset + limit, total);

  return (
    <View style={styles.row}>
      <Button label="Prev" onPress={onPrev} disabled={!canPrev} variant="secondary" />
      <Text variant="caption" color="secondary" style={styles.range}>
        {start}–{end} of {total}
      </Text>
      <Button label="Next" onPress={onNext} disabled={!canNext} variant="secondary" />
    </View>
  );
}
