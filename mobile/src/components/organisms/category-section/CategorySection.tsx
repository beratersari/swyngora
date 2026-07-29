import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import { SectionHeader } from '@/components/molecules/section-header';
import { formatCategoryLabel } from '@/libs/utils';
import type { CategorySectionProps } from './CategorySection.types';
import { styles } from './CategorySection.styles';

export function CategorySection({
  title,
  actionLabel,
  onAction,
  tags,
  onSelectTag,
  isLoading = false,
  errorMessage,
  emptyMessage,
  onRetry,
  retryLabel,
  formatLabel = formatCategoryLabel,
}: CategorySectionProps) {
  if (!isLoading && !errorMessage && tags.length === 0 && !emptyMessage) {
    return null;
  }

  return (
    <View style={styles.root}>
      <SectionHeader title={title} actionLabel={actionLabel} onAction={onAction} />
      {errorMessage ? (
        <View style={styles.errorRow}>
          <Text variant="caption" color="error">
            {errorMessage}
          </Text>
          {onRetry && retryLabel ? (
            <Pressable onPress={onRetry} accessibilityRole="button">
              <Text variant="caption" color="cream">
                {retryLabel}
              </Text>
            </Pressable>
          ) : null}
        </View>
      ) : (
        <ChipGroup
          options={tags.map((t) => ({ value: t, label: formatLabel(t) }))}
          selected=""
          onSelect={onSelectTag}
          mode="single"
          shape="pill"
          horizontalScroll
          isLoading={isLoading}
          emptyLabel={emptyMessage ?? undefined}
        />
      )}
    </View>
  );
}
