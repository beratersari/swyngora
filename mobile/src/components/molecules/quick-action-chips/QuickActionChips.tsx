import { View } from 'react-native';
import { Chip } from '@/components/molecules/chip';
import type { QuickActionChipsProps } from './QuickActionChips.types';
import { styles } from './QuickActionChips.styles';

export function QuickActionChips({ actions }: QuickActionChipsProps) {
  return (
    <View style={styles.row}>
      {actions.map((a) => (
        <Chip key={a.id} label={a.label} onPress={a.onPress} shape="pill" active={false} />
      ))}
    </View>
  );
}
