import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import { formatCategoryLabel } from '@/libs/utils';
import type { CategoryChipGridProps } from './CategoryChipGrid.types';
import { styles } from './CategoryChipGrid.styles';

export function CategoryChipGrid({
  featuredTitle,
  featuredTags,
  allTitle,
  tags,
  selectedTag = null,
  onSelectTag,
  isLoading = false,
  emptyMessage,
  formatLabel = formatCategoryLabel,
}: CategoryChipGridProps) {
  const selected = selectedTag ?? '';

  return (
    <View style={styles.root}>
      {featuredTags.length > 0 || isLoading ? (
        <View style={styles.block}>
          {featuredTitle ? (
            <Text variant="label" color="secondary" style={styles.title}>
              {featuredTitle}
            </Text>
          ) : null}
          <ChipGroup
            options={featuredTags.map((t) => ({ value: t, label: formatLabel(t) }))}
            selected={selected}
            onSelect={onSelectTag}
            mode="single"
            shape="pill"
            isLoading={isLoading && featuredTags.length === 0}
          />
        </View>
      ) : null}

      <View style={styles.block}>
        {allTitle ? (
          <Text variant="label" color="secondary" style={styles.title}>
            {allTitle}
          </Text>
        ) : null}
        <ChipGroup
          options={tags.map((t) => ({ value: t, label: formatLabel(t) }))}
          selected={selected}
          onSelect={onSelectTag}
          mode="single"
          shape="box"
          isLoading={isLoading && tags.length === 0}
          emptyLabel={emptyMessage ?? undefined}
        />
      </View>
    </View>
  );
}
