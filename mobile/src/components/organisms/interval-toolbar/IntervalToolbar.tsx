import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import type { IntervalToolbarProps } from './IntervalToolbar.types';
import { styles } from './IntervalToolbar.styles';

export function IntervalToolbar({
  intervals,
  selected,
  onSelect,
  isLoading,
  showEma,
  onToggleEma,
}: IntervalToolbarProps) {
  const { t } = useTranslation('detail');
  return (
    <View style={styles.root}>
      <Text variant="label" color="secondary">
        {t('interval')}
      </Text>
      <View style={styles.row}>
        <ChipGroup
          options={intervals.map((i) => ({ value: i, label: i }))}
          selected={selected}
          onSelect={onSelect}
          mode="single"
          shape="box"
          horizontalScroll
          isLoading={isLoading}
        />
        <ChipGroup
          options={[
            { value: 'ema', label: showEma ? t('emaOn') : t('emaOff') },
          ]}
          selected={showEma ? 'ema' : ''}
          onSelect={onToggleEma}
          mode="single"
          shape="box"
        />
      </View>
    </View>
  );
}
