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
  showPumps = false,
  onTogglePumps,
  showPumpMargin = false,
  onTogglePumpMargin,
}: IntervalToolbarProps) {
  const { t } = useTranslation('detail');
  const overlayOptions = [
    { value: 'ema', label: showEma ? t('emaOn') : t('emaOff') },
    ...(onTogglePumps
      ? [{ value: 'pumps', label: showPumps ? t('pumpsOn') : t('pumpsOff') }]
      : []),
    ...(onTogglePumpMargin
      ? [
          {
            value: 'margin',
            label: showPumpMargin ? t('marginOn') : t('marginOff'),
          },
        ]
      : []),
  ];
  const overlaySelected = [
    ...(showEma ? ['ema'] : []),
    ...(showPumps && onTogglePumps ? ['pumps'] : []),
    ...(showPumpMargin && onTogglePumpMargin ? ['margin'] : []),
  ];

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
      </View>
      <Text variant="label" color="secondary">
        {t('chartOverlays')}
      </Text>
      <View style={styles.row}>
        <ChipGroup
          options={overlayOptions}
          selected={overlaySelected}
          onSelect={(value) => {
            if (value === 'ema') onToggleEma();
            else if (value === 'pumps') onTogglePumps?.();
            else if (value === 'margin') onTogglePumpMargin?.();
          }}
          mode="multi"
          shape="box"
        />
      </View>
    </View>
  );
}
