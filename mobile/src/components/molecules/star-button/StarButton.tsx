import { Pressable } from 'react-native';
import { Star } from 'lucide-react-native';
import { Icon, ICON_FAVORITE_GOLD } from '@/components/atoms/icon';
import { semanticColors } from '@/styles/tokens';
import type { StarButtonProps } from './StarButton.types';
import { styles } from './StarButton.styles';

export function StarButton({
  watched,
  onPress,
  disabled = false,
  accessibilityLabel,
  size = 'md',
}: StarButtonProps) {
  const idle = semanticColors.text.secondary;
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
      <Icon
        icon={Star}
        size={size === 'sm' ? 'sm' : 'md'}
        color={watched ? ICON_FAVORITE_GOLD : idle}
        fill={watched ? ICON_FAVORITE_GOLD : 'transparent'}
        strokeWidth={watched ? 1.5 : 2}
      />
    </Pressable>
  );
}
