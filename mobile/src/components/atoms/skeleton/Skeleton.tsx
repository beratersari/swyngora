import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import type { SkeletonProps } from './Skeleton.types';
import { styles } from './Skeleton.styles';

export function Skeleton({
  width = '100%',
  height = 16,
  borderRadius,
}: SkeletonProps) {
  const { t } = useTranslation('common');
  return (
    <View
      accessibilityLabel={t('status.loading')}
      accessibilityRole="progressbar"
      style={[
        styles.base,
        {
          width,
          height,
          ...(borderRadius !== undefined ? { borderRadius } : null),
        },
      ]}
    />
  );
}
