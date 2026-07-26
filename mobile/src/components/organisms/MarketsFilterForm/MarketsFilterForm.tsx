import { Pressable, ScrollView, View } from 'react-native';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { ChipGroup } from '@/components/molecules/ChipGroup';
import { SearchField } from '@/components/molecules/SearchField';
import type { MarketsFilterFormProps } from './MarketsFilterForm.types';
import { styles } from './MarketsFilterForm.styles';

export function MarketsFilterForm({
  quote,
  quoteOptions,
  onQuoteChange,
  sort,
  order,
  sortOptions,
  onSortChange,
  onOrderChange,
  availableTags,
  selectedTags,
  isLoadingTags,
  tagsError,
  tagSearch,
  onTagSearchChange,
  onToggleTag,
  onClearTags,
  onSelectAllVisible,
  onResetAll,
}: MarketsFilterFormProps) {
  return (
    <View style={styles.root}>
      <View style={styles.section}>
        <Text variant="label" color="secondary" style={styles.sectionTitle}>
          Quote
        </Text>
        <ChipGroup
          options={quoteOptions.map((q) => ({ value: q, label: q }))}
          selected={quote}
          onSelect={onQuoteChange}
          mode="single"
          shape="box"
        />
      </View>

      <View style={styles.section}>
        <Text variant="label" color="secondary" style={styles.sectionTitle}>
          Sort
        </Text>
        <View style={styles.actions}>
          <ChipGroup
            options={sortOptions}
            selected={sort}
            onSelect={onSortChange}
            mode="single"
            shape="box"
          />
          <ChipGroup
            options={[
              {
                value: order,
                label: order === 'desc' ? 'Desc' : 'Asc',
              },
            ]}
            selected={order}
            onSelect={() => onOrderChange(order === 'desc' ? 'asc' : 'desc')}
            mode="single"
            shape="box"
          />
        </View>
      </View>

      <View style={styles.section}>
        <Text variant="label" color="secondary" style={styles.sectionTitle}>
          Tags {selectedTags.length > 0 ? `(${selectedTags.length})` : ''}
        </Text>
        <SearchField
          value={tagSearch}
          onChangeText={onTagSearchChange}
          placeholder="Search tags…"
          accessibilityLabel="Search tags"
        />
        <View style={styles.actions}>
          <Pressable onPress={onClearTags}>
            <Text variant="caption" color="steel">
              Clear tags
            </Text>
          </Pressable>
          <Pressable onPress={onSelectAllVisible}>
            <Text variant="caption" color="steel">
              Select visible
            </Text>
          </Pressable>
          <Pressable onPress={onResetAll}>
            <Text variant="caption" color="steel">
              Reset all filters
            </Text>
          </Pressable>
        </View>

        {isLoadingTags ? (
          <View style={styles.skeletonRow}>
            {Array.from({ length: 12 }).map((_, i) => (
              <Skeleton key={i} width={88} height={32} borderRadius={999} />
            ))}
          </View>
        ) : tagsError ? (
          <Text variant="body" color="error">
            {tagsError}
          </Text>
        ) : (
          <ScrollView style={styles.tagsScroll} contentContainerStyle={styles.tagRow}>
            <ChipGroup
              options={availableTags.map((t) => ({ value: t, label: t }))}
              selected={selectedTags}
              onSelect={onToggleTag}
              mode="multi"
              shape="pill"
              emptyLabel="No tags match"
            />
          </ScrollView>
        )}
      </View>
    </View>
  );
}
