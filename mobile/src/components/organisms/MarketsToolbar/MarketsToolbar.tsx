import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { SearchField } from '@/components/molecules/SearchField';
import type { MarketsToolbarProps } from './MarketsToolbar.types';
import { styles } from './MarketsToolbar.styles';

export function MarketsToolbar({
  search,
  onSearchChange,
  isSearchDebouncing = false,
  activeFilterCount,
  onOpenFilters,
}: MarketsToolbarProps) {
  const hasFilters = activeFilterCount > 0;
  return (
    <View style={styles.root}>
      <View style={styles.row}>
        <SearchField
          value={search}
          onChangeText={onSearchChange}
          placeholder="Search pairs…"
          accessibilityLabel="Search markets"
          autoCapitalize="characters"
          style={styles.search}
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Open filters"
          onPress={onOpenFilters}
          style={[styles.filterBtn, hasFilters && styles.filterBtnActive]}
        >
          <Text variant="label" color={hasFilters ? 'cream' : 'secondary'}>
            {hasFilters ? `Filters (${activeFilterCount})` : 'Filters'}
          </Text>
        </Pressable>
      </View>
      {isSearchDebouncing ? (
        <Text variant="caption" color="steel">
          Searching…
        </Text>
      ) : null}
    </View>
  );
}
