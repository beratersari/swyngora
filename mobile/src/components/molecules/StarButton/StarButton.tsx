import { Pressable, Text as RNText } from 'react-native';
import type { StarButtonProps } from './StarButton.types';
import { styles } from './StarButton.styles';

export function StarButton({
  watched,
  onPress,
  disabled = false,
  accessibilityLabel,
  size = 'md',
}: StarButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: watched, disabled }}
      accessibilityLabel={
        accessibilityLabel ??
        (watched ? 'Remove from favorites' : 'Add to favorites')
      }
      disabled={disabled}
      onPress={(e) => {
        e?.stopPropagation?.();
        onPress();
      }}
      style={[styles.hit, size === 'sm' && styles.hitSm]}
      hitSlop={8}
    >
      <RNText
        style={[
          size === 'sm' ? styles.starSm : styles.star,
          watched ? styles.watched : styles.idle,
        ]}
      >
        {watched ? '★' : '☆'}
      </RNText>
    </Pressable>
  );
}
