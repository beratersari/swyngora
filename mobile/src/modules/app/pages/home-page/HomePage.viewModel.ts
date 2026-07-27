import { HEALTH_POLL_MS } from '@/config/constants';
import { env } from '@/config/env';
import { useGetHealthQuery, rtkErrorMessage } from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import type { HomePageViewModel } from './HomePage.types';

export function useHomePageViewModel(): HomePageViewModel {
  const active = useAppStateActive();
  const healthQuery = useGetHealthQuery(undefined, {
    pollingInterval: active ? HEALTH_POLL_MS : 0,
    refetchOnFocus: false,
  });

  let healthStatus: HomePageViewModel['healthStatus'] = 'unknown';
  if (healthQuery.isSuccess) healthStatus = 'ok';
  else if (healthQuery.isError) healthStatus = 'error';

  return {
    title: 'Swyngora',
    apiBaseUrlLabel: env.apiBaseUrlLabel,
    healthStatus,
    healthDetail: healthQuery.data?.status ?? healthQuery.data?.time ?? null,
    isLoading: healthQuery.isLoading || healthQuery.isFetching,
    isPollingPaused: !active,
    errorMessage: healthQuery.isError
      ? rtkErrorMessage(healthQuery.error, { resource: 'health' })
      : null,
    onRetry: () => {
      void healthQuery.refetch();
    },
  };
}
