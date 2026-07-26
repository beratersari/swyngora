import { View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Text } from '@/components/atoms/Text';
import type { ScreenTemplateProps } from './ScreenTemplate.types';
import { styles } from './ScreenTemplate.styles';

export function ScreenTemplate({ title, children, footer }: ScreenTemplateProps) {
  return (
    <SafeAreaView style={styles.safe} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Text variant="h2">{title}</Text>
      </View>
      <View style={styles.content}>{children}</View>
      {footer ? <View style={styles.footer}>{footer}</View> : null}
    </SafeAreaView>
  );
}
