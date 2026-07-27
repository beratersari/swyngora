import { View } from 'react-native';
import type { SkeletonProps } from './Skeleton.types';
import { styles } from './Skeleton.styles';

export function Skeleton({
  width = '100%',
  height = 16,
  borderRadius,
}: SkeletonProps) {
  return (
    <View
      accessibilityLabel="Loading"
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
