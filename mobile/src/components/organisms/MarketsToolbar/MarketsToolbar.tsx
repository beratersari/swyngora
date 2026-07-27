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
  favoritesOnly = false,
  onToggleFavoritesOnly,
  favoritesCount = 0,
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
      {onToggleFavoritesOnly ? (
        <View style={styles.row}>
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ selected: favoritesOnly }}
            accessibilityLabel={
              favoritesOnly ? 'Show all markets' : 'Show favorites only'
            }
            onPress={onToggleFavoritesOnly}
            style={[styles.favBtn, favoritesOnly && styles.favBtnActive]}
          >
            <Text variant="label" color={favoritesOnly ? 'cream' : 'secondary'}>
              {favoritesOnly
                ? `★ Favorites only (${favoritesCount})`
                : `☆ Favorites${favoritesCount > 0 ? ` (${favoritesCount})` : ''}`}
            </Text>
          </Pressable>
        </View>
      ) : null}
      {isSearchDebouncing ? (
        <Text variant="caption" color="steel">
          Searching…
        </Text>
      ) : null}
    </View>
  );
}
