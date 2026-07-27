import { useCallback, useMemo, useState } from 'react';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import {
  rtkErrorMessage,
  useListProductTagsQuery,
  type SpotSortField,
  type SpotSortOrder,
} from '@/libs/api';
import { useDebouncedValue } from '@/libs/hooks';
import { isSpotSortField } from '@/libs/utils';
import { useMarketsContext } from '../../context';
import type { MarketsStackParamList } from '../../navigation';
import {
  DEFAULT_ORDER,
  DEFAULT_QUOTE,
  DEFAULT_SORT,
  QUOTE_OPTIONS,
  SORT_OPTIONS,
} from '../markets-page/MarketsPage.constants';
import type { MarketsFilterPageViewModel } from './MarketsFilterPage.types';

const TAG_SEARCH_DEBOUNCE_MS = 200;

export function useMarketsFilterPageViewModel(): MarketsFilterPageViewModel {
  const navigation = useNavigation<NativeStackNavigationProp<MarketsStackParamList>>();
  const markets = useMarketsContext();
  const tagsQuery = useListProductTagsQuery({ exchange: markets.exchange });

  const [quote, setQuote] = useState(markets.quote);
  const [sort, setSort] = useState<SpotSortField>(markets.sort);
  const [order, setOrder] = useState<SpotSortOrder>(markets.order);
  const [draftTags, setDraftTags] = useState<string[]>(() => [...markets.selectedTags]);
  const [searchTag, setSearchTag] = useState('');
  const debouncedTagSearch = useDebouncedValue(searchTag, TAG_SEARCH_DEBOUNCE_MS);

  const visibleTags = useMemo(() => {
    const availableTags = tagsQuery.data?.tags ?? [];
    const q = debouncedTagSearch.trim().toLowerCase();
    if (!q) return availableTags;
    return availableTags.filter((t) => t.toLowerCase().includes(q));
  }, [tagsQuery.data?.tags, debouncedTagSearch]);

  const onQuoteChange = useCallback((next: string) => {
    setQuote(next);
  }, []);

  const onSortChange = useCallback((next: string) => {
    if (isSpotSortField(next)) setSort(next);
  }, []);

  const onOrderChange = useCallback((next: 'asc' | 'desc') => {
    setOrder(next);
  }, []);

  const onToggleTag = useCallback((tag: string) => {
    setDraftTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  }, []);

  const onClearTags = useCallback(() => {
    setDraftTags([]);
  }, []);

  const onSelectAllVisible = useCallback(() => {
    setDraftTags((prev) => {
      const set = new Set(prev);
      for (const t of visibleTags) set.add(t);
      return [...set];
    });
  }, [visibleTags]);

  const onResetAll = useCallback(() => {
    setQuote(DEFAULT_QUOTE);
    setSort(DEFAULT_SORT);
    setOrder(DEFAULT_ORDER);
    setDraftTags([]);
  }, []);

  const onApply = useCallback(() => {
    markets.applyListFilters({
      quote,
      sort,
      order,
      selectedTags: draftTags,
    });
    navigation.goBack();
  }, [markets, quote, sort, order, draftTags, navigation]);

  const onCancel = useCallback(() => {
    navigation.goBack();
  }, [navigation]);

  return {
    title: 'Filters',
    quote,
    quoteOptions: [...QUOTE_OPTIONS],
    onQuoteChange,
    sort,
    order,
    sortOptions: SORT_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
    onSortChange,
    onOrderChange,
    availableTags: visibleTags,
    draftTags,
    isLoadingTags: tagsQuery.isLoading,
    tagsError: tagsQuery.isError
      ? rtkErrorMessage(tagsQuery.error, { resource: 'tags' })
      : null,
    searchTag,
    onSearchTagChange: setSearchTag,
    onToggleTag,
    onClearTags,
    onSelectAllVisible,
    onResetAll,
    onApply,
    onCancel,
    selectedTagsCount: draftTags.length,
  };
}
