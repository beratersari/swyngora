import { Pressable, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { LayoutGrid, ListOrdered, SlidersHorizontal, Star, X } from 'lucide-react-native';
import { Icon, ICON_FAVORITE_GOLD } from '@/components/atoms/icon';
import { Text } from '@/components/atoms/text';
import { SearchField } from '@/components/molecules/search-field';
import { colors, semanticColors } from '@/styles/tokens';
import type { MarketsToolbarProps } from './MarketsToolbar.types';
import { styles } from './MarketsToolbar.styles';

export function MarketsToolbar({
  search,
  onSearchChange,
  isSearchDebouncing = false,
  activeFilterCount,
  onOpenFilters,
  onOpenCategories,
  categoriesLabel,
  categoriesA11y,
  onOpenLeaderboards,
  leaderboardsLabel,
  leaderboardsA11y,
  activeCategoryLabel,
  onClearCategory,
  clearCategoryA11y,
  favoritesOnly = false,
  onToggleFavoritesOnly,
  favoritesCount = 0,
}: MarketsToolbarProps) {
  const { t } = useTranslation(['markets', 'common']);
  const hasFilters = activeFilterCount > 0;
  const filterIconColor = hasFilters
    ? colors.cream
    : semanticColors.text.secondary;
  const favIconColor = favoritesOnly
    ? ICON_FAVORITE_GOLD
    : semanticColors.text.secondary;
  const catLabel = categoriesLabel ?? t('markets:categories');
  const catA11y = categoriesA11y ?? t('markets:openCategoriesA11y');
  const boardsLabel = leaderboardsLabel ?? t('markets:leaderboards');
  const boardsA11y = leaderboardsA11y ?? t('markets:openLeaderboardsA11y');
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
        {onOpenLeaderboards ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={boardsA11y}
            onPress={onOpenLeaderboards}
            style={styles.filterBtn}
          >
            <View style={styles.btnInner}>
              <Icon icon={ListOrdered} size="sm" color={semanticColors.text.secondary} />
              <Text variant="label" color="secondary">
                {boardsLabel}
              </Text>
            </View>
          </Pressable>
        ) : null}
        {onOpenCategories ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={catA11y}
            onPress={onOpenCategories}
            style={styles.filterBtn}
          >
            <View style={styles.btnInner}>
              <Icon icon={LayoutGrid} size="sm" color={semanticColors.text.secondary} />
              <Text variant="label" color="secondary">
                {catLabel}
              </Text>
            </View>
          </Pressable>
        ) : null}
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t('markets:openFiltersA11y')}
          onPress={onOpenFilters}
          style={[styles.filterBtn, hasFilters && styles.filterBtnActive]}
        >
          <View style={styles.btnInner}>
            <Icon icon={SlidersHorizontal} size="sm" color={filterIconColor} />
            <Text variant="label" color={hasFilters ? 'cream' : 'secondary'}>
              {hasFilters
                ? t('markets:filtersWithCount', { count: activeFilterCount })
                : t('markets:filters')}
            </Text>
          </View>
        </Pressable>
      </View>
      {activeCategoryLabel ? (
        <View style={styles.row}>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={
              clearCategoryA11y ?? t('markets:clearCategoryA11y')
            }
            onPress={onClearCategory}
            style={[styles.favBtn, styles.categoryChip]}
          >
            <View style={styles.btnInner}>
              <Text variant="label" color="cream">
                {activeCategoryLabel}
              </Text>
              {onClearCategory ? (
                <Icon icon={X} size="sm" color={colors.cream} />
              ) : null}
            </View>
          </Pressable>
        </View>
      ) : null}
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
            <View style={styles.btnInner}>
              <Icon
                icon={Star}
                size="sm"
                color={favIconColor}
                fill={favoritesOnly ? ICON_FAVORITE_GOLD : 'transparent'}
                strokeWidth={favoritesOnly ? 1.5 : 2}
              />
              <Text
                variant="label"
                color={favoritesOnly ? 'cream' : 'secondary'}
              >
                {favoritesOnly
                  ? t('markets:favoritesOnly', { count: favoritesCount })
                  : favoritesCount > 0
                    ? t('markets:favoritesToggleWithCount', {
                        count: favoritesCount,
                      })
                    : t('markets:favoritesToggle')}
              </Text>
            </View>
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
