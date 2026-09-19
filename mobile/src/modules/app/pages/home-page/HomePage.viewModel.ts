import { useCallback, useMemo } from 'react';
import { useIsFocused, useNavigation } from '@react-navigation/native';
import { useTranslation } from 'react-i18next';
import {
  HOME_DEFAULT_EXCHANGE,
  HOME_FAVORITES_LIMIT,
  HOME_WIDGET_POLL_MS,
} from '@/config/homeDashboardConstants';
import { CATEGORY_TAGS_EXCHANGE } from '@/config/categoryConstants';
import { env } from '@/config/env';
import {
  rtkErrorMessage,
  useGetHealthQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useScanPumpEventsQuery,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  buildHomePumpScanQuery,
  buildMoversSpotQuery,
  buildVolumeSpotQuery,
  formatCategoryLabel,
  indexDashboardRows,
  intersectFeaturedTags,
  mapFavoritesToDashboardRows,
  mapPumpHitsToTeasers,
  mapSpotListToDashboardRows,
} from '@/libs/utils';
import { MarketsScreens, useOptionalMarketsContext } from '@/modules/markets';
import { PumpsScreens } from '@/modules/pumps';
import { useOptionalWatchlist } from '@/modules/watchlist';
import type { HomePageViewModel } from './HomePage.types';

export function useHomePageViewModel(): HomePageViewModel {
  const { t } = useTranslation(['home', 'common', 'markets']);
  const navigation = useNavigation();
  const active = useAppStateActive();
  const focused = useIsFocused();
  const watchlist = useOptionalWatchlist();
  const markets = useOptionalMarketsContext();

  const poll = active && focused ? HOME_WIDGET_POLL_MS : 0;
  const skipWidgets = !active || !focused;

  const moversArgs = useMemo(() => buildMoversSpotQuery(), []);
  const volumeArgs = useMemo(() => buildVolumeSpotQuery(), []);
  const pumpArgs = useMemo(() => buildHomePumpScanQuery(), []);

  const moversQuery = useListSpotMarketsQuery(moversArgs, {
    skip: skipWidgets,
    pollingInterval: poll,
    refetchOnFocus: false,
  });
  const volumeQuery = useListSpotMarketsQuery(volumeArgs, {
    skip: skipWidgets,
    pollingInterval: poll,
    refetchOnFocus: false,
  });
  const pumpsQuery = useScanPumpEventsQuery(pumpArgs, {
    skip: skipWidgets,
    pollingInterval: 0,
    refetchOnFocus: false,
  });
  const tagsQuery = useListProductTagsQuery(
    { exchange: CATEGORY_TAGS_EXCHANGE },
    {
      skip: skipWidgets,
      pollingInterval: 0,
      refetchOnFocus: false,
    },
  );
  const healthQuery = useGetHealthQuery(undefined, {
    pollingInterval: poll || 0,
    refetchOnFocus: false,
  });

  const exchange = HOME_DEFAULT_EXCHANGE;

  const movers = useMemo(
    () => mapSpotListToDashboardRows(moversQuery.data?.items, exchange),
    [moversQuery.data?.items, exchange],
  );
  const volume = useMemo(
    () => mapSpotListToDashboardRows(volumeQuery.data?.items, exchange),
    [volumeQuery.data?.items, exchange],
  );
  const spotIndex = useMemo(
    () => indexDashboardRows([...movers, ...volume]),
    [movers, volume],
  );

  const favorites = useMemo(() => {
    const pairs = watchlist?.items ?? [];
    return mapFavoritesToDashboardRows(pairs, HOME_FAVORITES_LIMIT, spotIndex);
  }, [watchlist?.items, spotIndex]);

  const pumps = useMemo(
    () => mapPumpHitsToTeasers(pumpsQuery.data?.hits, exchange),
    [pumpsQuery.data?.hits, exchange],
  );

  const navigateTab = useCallback(
    (name: string, params?: object) => {
      const parent = navigation.getParent() as
        | { navigate: (n: string, p?: object) => void }
        | undefined;
      parent?.navigate(name, params);
    },
    [navigation],
  );

  const onOpenMarkets = useCallback(() => {
    navigateTab('MarketsTab');
  }, [navigateTab]);

  const onOpenGainers = useCallback(() => {
    navigateTab('MarketsTab', {
      screen: MarketsScreens.Leaderboards,
      params: { board: 'gainers' },
    });
  }, [navigateTab]);

  const onOpenVolumeBoard = useCallback(() => {
    navigateTab('MarketsTab', {
      screen: MarketsScreens.Leaderboards,
      params: { board: 'volume' },
    });
  }, [navigateTab]);

  const onOpenLeaderboards = useCallback(() => {
    navigateTab('MarketsTab', {
      screen: MarketsScreens.Leaderboards,
      params: { board: 'gainers' },
    });
  }, [navigateTab]);

  const onOpenPumps = useCallback(() => {
    navigateTab('PumpsTab');
  }, [navigateTab]);

  const onOpenAsk = useCallback(() => {
    navigateTab('AskTab');
  }, [navigateTab]);

  const onOpenFavorites = useCallback(() => {
    navigateTab('WatchlistTab');
  }, [navigateTab]);

  const onOpenCategories = useCallback(() => {
    navigateTab('MarketsTab', {
      screen: MarketsScreens.Categories,
    });
  }, [navigateTab]);

  const onSelectCategory = useCallback(
    (tag: string) => {
      markets?.selectCategoryTag(tag);
      navigateTab('MarketsTab', {
        screen: MarketsScreens.List,
      });
    },
    [markets, navigateTab],
  );

  const onPressMarket = useCallback(
    (ex: string, symbol: string) => {
      navigateTab('MarketsTab', {
        screen: MarketsScreens.Detail,
        params: { exchange: ex, symbol },
      });
    },
    [navigateTab],
  );

  const onPressPump = useCallback(
    (ex: string, symbol: string) => {
      navigateTab('PumpsTab', {
        screen: PumpsScreens.Detail,
        params: { exchange: ex, symbol },
      });
    },
    [navigateTab],
  );

  const onRefresh = useCallback(() => {
    void moversQuery.refetch();
    void volumeQuery.refetch();
    void pumpsQuery.refetch();
    void tagsQuery.refetch();
    void healthQuery.refetch();
  }, [moversQuery, volumeQuery, pumpsQuery, tagsQuery, healthQuery]);

  let healthStatus: HomePageViewModel['healthStatus'] = 'unknown';
  if (healthQuery.isSuccess) healthStatus = 'ok';
  else if (healthQuery.isError) healthStatus = 'error';

  const isRefreshing =
    (moversQuery.isFetching || volumeQuery.isFetching || pumpsQuery.isFetching) &&
    (movers.length > 0 || volume.length > 0 || pumps.length > 0);

  const liveTags = tagsQuery.data?.tags ?? [];
  const categoryTags = useMemo(() => intersectFeaturedTags(liveTags), [liveTags]);
  const categoriesLoading =
    tagsQuery.isLoading || (tagsQuery.isFetching && liveTags.length === 0);
  const categoriesError = tagsQuery.isError
    ? rtkErrorMessage(tagsQuery.error, { resource: 'tags' })
    : null;
  const categoriesEmpty =
    !categoriesError && !categoriesLoading && categoryTags.length === 0
      ? t('home:categoriesEmpty')
      : null;

  return {
    title: t('home:title'),
    intro: t('home:intro'),
    quickActions: [
      { id: 'markets', label: t('home:openMarkets'), onPress: onOpenMarkets },
      { id: 'leaderboards', label: t('home:openLeaderboards'), onPress: onOpenLeaderboards },
      { id: 'pumps', label: t('home:openPumps'), onPress: onOpenPumps },
      { id: 'ask', label: t('home:openAsk'), onPress: onOpenAsk },
    ],

    favorites,
    favoritesLoading: Boolean(watchlist && !watchlist.isReady),
    favoritesEmpty:
      watchlist?.isReady && favorites.length === 0
        ? t('home:favoritesEmpty')
        : null,
    favoritesTitle: t('home:sectionFavorites'),

    movers,
    moversLoading: moversQuery.isLoading || (moversQuery.isFetching && movers.length === 0),
    moversError: moversQuery.isError
      ? rtkErrorMessage(moversQuery.error, { resource: 'movers' })
      : null,
    moversEmpty:
      !moversQuery.isError && !moversQuery.isLoading && movers.length === 0
        ? t('home:moversEmpty')
        : null,
    moversTitle: t('home:sectionMovers'),
    onOpenMoversSeeAll: onOpenGainers,
    onRetryMovers: () => {
      void moversQuery.refetch();
    },

    volume,
    volumeLoading: volumeQuery.isLoading || (volumeQuery.isFetching && volume.length === 0),
    volumeError: volumeQuery.isError
      ? rtkErrorMessage(volumeQuery.error, { resource: 'volume' })
      : null,
    volumeEmpty:
      !volumeQuery.isError && !volumeQuery.isLoading && volume.length === 0
        ? t('home:volumeEmpty')
        : null,
    volumeTitle: t('home:sectionVolume'),
    onOpenVolumeSeeAll: onOpenVolumeBoard,
    onRetryVolume: () => {
      void volumeQuery.refetch();
    },

    pumps,
    pumpsLoading: pumpsQuery.isLoading || (pumpsQuery.isFetching && pumps.length === 0),
    pumpsError: pumpsQuery.isError
      ? rtkErrorMessage(pumpsQuery.error, { resource: 'pumps' })
      : null,
    pumpsEmpty:
      !pumpsQuery.isError && !pumpsQuery.isLoading && pumps.length === 0
        ? t('home:pumpsEmpty')
        : null,
    pumpsTitle: t('home:sectionPumps'),
    pumpsDisclaimer: pumpsQuery.data?.note ?? t('home:pumpsHint'),
    onRetryPumps: () => {
      void pumpsQuery.refetch();
    },

    categoriesTitle: t('home:sectionCategories'),
    categoryTags,
    categoriesLoading,
    categoriesError,
    categoriesEmpty,
    onSelectCategory,
    onOpenCategories,
    onRetryCategories: () => {
      void tagsQuery.refetch();
    },
    formatCategoryLabel,

    seeAllLabel: t('home:seeAll'),
    retryLabel: t('common:actions.retry'),
    isRefreshing,
    isPollingPaused: !active || !focused,
    pollingCaption: t('home:polling', {
      state:
        !active || !focused
          ? t('common:status.pollingPaused')
          : t('common:status.pollingActive'),
    }),

    healthStatus,
    healthDetail: healthQuery.data?.status ?? healthQuery.data?.time ?? null,
    apiBaseUrlLabel: env.apiBaseUrlLabel,

    onRefresh,
    onOpenMarkets,
    onOpenPumps,
    onOpenAsk,
    onPressMarket,
    onPressPump,
    onOpenFavorites,
  };
}
