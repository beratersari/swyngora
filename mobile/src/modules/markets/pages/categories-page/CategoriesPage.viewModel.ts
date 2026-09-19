import { useCallback, useMemo, useState } from 'react';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { useTranslation } from 'react-i18next';
import {
  CATEGORY_TAG_SEARCH_DEBOUNCE_MS,
  CATEGORY_TAGS_EXCHANGE,
} from '@/config/categoryConstants';
import { rtkErrorMessage, useListProductTagsQuery } from '@/libs/api';
import { useDebouncedValue } from '@/libs/hooks';
import {
  filterTagsBySearch,
  formatCategoryLabel,
  intersectFeaturedTags,
} from '@/libs/utils';
import { useMarketsContext } from '../../context';
import { MarketsScreens, type MarketsStackParamList } from '../../navigation';
import type { CategoriesPageViewModel } from './CategoriesPage.types';

export function useCategoriesPageViewModel(): CategoriesPageViewModel {
  const { t } = useTranslation(['markets', 'common']);
  const navigation =
    useNavigation<NativeStackNavigationProp<MarketsStackParamList>>();
  const markets = useMarketsContext();

  const tagsQuery = useListProductTagsQuery({ exchange: CATEGORY_TAGS_EXCHANGE });

  const [search, setSearch] = useState('');
  const debouncedSearch = useDebouncedValue(search, CATEGORY_TAG_SEARCH_DEBOUNCE_MS);
  const isSearchDebouncing = search.trim() !== debouncedSearch.trim();

  const liveTags = tagsQuery.data?.tags ?? [];
  const featuredTags = useMemo(() => intersectFeaturedTags(liveTags), [liveTags]);
  const filteredTags = useMemo(
    () => filterTagsBySearch(liveTags, debouncedSearch),
    [liveTags, debouncedSearch],
  );

  const selectedTag =
    markets.selectedTags.length === 1 ? markets.selectedTags[0] : null;

  const onSelectTag = useCallback(
    (tag: string) => {
      markets.selectCategoryTag(tag);
      navigation.navigate(MarketsScreens.List);
    },
    [markets, navigation],
  );

  const onBack = useCallback(() => {
    if (navigation.canGoBack()) {
      navigation.goBack();
    } else {
      navigation.navigate(MarketsScreens.List);
    }
  }, [navigation]);

  const isLoading = tagsQuery.isLoading || (tagsQuery.isFetching && liveTags.length === 0);

  const errorMessage = tagsQuery.isError
    ? rtkErrorMessage(tagsQuery.error, { resource: 'tags' })
    : null;

  const emptyMessage =
    !errorMessage && !isLoading && !isSearchDebouncing && filteredTags.length === 0
      ? debouncedSearch.trim()
        ? t('markets:noTagsMatch')
        : t('markets:categoriesEmpty')
      : null;

  return {
    title: t('markets:categoriesTitle'),
    search,
    onSearchChange: setSearch,
    searchPlaceholder: t('markets:searchTagsPlaceholder'),
    featuredTitle: t('markets:categoriesFeatured'),
    featuredTags: debouncedSearch.trim() ? [] : featuredTags,
    allTitle: t('markets:categoriesAll'),
    tags: filteredTags,
    selectedTag,
    isLoading,
    isSearchDebouncing,
    errorMessage,
    emptyMessage,
    formatLabel: formatCategoryLabel,
    onSelectTag,
    onRetry: () => {
      void tagsQuery.refetch();
    },
    onBack,
    retryLabel: t('common:actions.retry'),
    backLabel: t('common:actions.back', { defaultValue: 'Back' }),
  };
}
