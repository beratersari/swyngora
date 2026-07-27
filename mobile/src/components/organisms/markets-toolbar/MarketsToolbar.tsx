import { Pressable, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { SearchField } from '@/components/molecules/search-field';
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
  const { t } = useTranslation(['markets', 'common']);
  const hasFilters = activeFilterCount > 0;
  return (
    <View style={styles.root}>
      <View style={styles.row}>
        <SearchField
          value={search}
          onChangeText={onSearchChange}
          placeholder={t('markets:searchPlaceholder')}
          accessibilityLabel={t('markets:searchA11y')}
          autoCapitalize="characters"
          style={styles.search}
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t('markets:openFiltersA11y')}
          onPress={onOpenFilters}
          style={[styles.filterBtn, hasFilters && styles.filterBtnActive]}
        >
          <Text variant="label" color={hasFilters ? 'cream' : 'secondary'}>
            {hasFilters
              ? t('markets:filtersWithCount', { count: activeFilterCount })
              : t('markets:filters')}
          </Text>
        </Pressable>
      </View>
      {onToggleFavoritesOnly ? (
        <View style={styles.row}>
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ selected: favoritesOnly }}
            accessibilityLabel={
              favoritesOnly
                ? t('markets:showAllMarketsA11y')
                : t('markets:showFavoritesOnlyA11y')
            }
            onPress={onToggleFavoritesOnly}
            style={[styles.favBtn, favoritesOnly && styles.favBtnActive]}
          >
            <Text variant="label" color={favoritesOnly ? 'cream' : 'secondary'}>
              {favoritesOnly
                ? t('markets:favoritesOnly', { count: favoritesCount })
                : favoritesCount > 0
                  ? t('markets:favoritesToggleWithCount', { count: favoritesCount })
                  : t('markets:favoritesToggle')}
            </Text>
          </Pressable>
        </View>
      ) : null}
      {isSearchDebouncing ? (
        <Text variant="caption" color="steel">
          {t('common:status.searching')}
        </Text>
      ) : null}
    </View>
  );
}
