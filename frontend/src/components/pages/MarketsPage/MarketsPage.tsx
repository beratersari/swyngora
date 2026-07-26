import { Alert, Tag } from 'antd';
import { useGetHealthQuery } from '@/libs/api';
import { CandleChartHost } from '@/components/molecules/CandleChartHost';
import { Text } from '@/components/atoms/Text';
import { Skeleton } from '@/components/atoms/Skeleton';
import { APP_NAME } from '@/config/constants';
import { env } from '@/config/env';
import { BlockSpacer, PageIntro, PageStack, PanelCard, StatusRow } from './MarketsPage.styles';

/** Placeholder Markets route — full spot table is Epic B. */
export function MarketsPage() {
  const { data, error, isLoading, isFetching } = useGetHealthQuery(undefined, {
    pollingInterval: 30_000,
  });

  const apiOk = data?.status === 'ok';
  const statusLoading = isLoading && !data;

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="cream">
          Markets
        </Text>
        <Text variant="body" color="steel">
          Multi-exchange spot markets UI ships in Epic B. This shell confirms the app, design system
          (styled-components), Ant Design, RTK Query, and Lightweight Charts.
        </Text>
      </PageIntro>

      <PanelCard
        title={
          <Text variant="h4" color="cream" as="span">
            API status
          </Text>
        }
        size="small"
      >
        <StatusRow>
          <Text variant="bodySm" color="steel" as="span">
            API:{' '}
            <Text
              variant="code"
              color="cream"
              as="span"
              isLoading={statusLoading}
              skeletonWidth={140}
            >
              {env.apiBaseUrlLabel}
            </Text>
          </Text>
          {isLoading || isFetching ? <Tag color="processing">checking…</Tag> : null}
          {apiOk ? <Tag color="success">backend ok</Tag> : null}
          {error ? <Tag color="error">backend unreachable</Tag> : null}
        </StatusRow>
        {error ? (
          <BlockSpacer>
            <Alert
              type="warning"
              showIcon
              message="Cannot reach backend"
              description={
                'Start the API in WSL: cd backend && go run ./cmd/server. ' +
                'Dev UI uses a Vite proxy to 127.0.0.1:8080 — restart `npm run dev` after .env changes. ' +
                'Do not set VITE_API_BASE_URL=http://localhost:8080 when opening the app via a WSL IP from Windows.'
              }
            />
          </BlockSpacer>
        ) : null}
        {statusLoading ? (
          <BlockSpacer>
            <Skeleton variant="text" rows={2} active />
          </BlockSpacer>
        ) : null}
        {apiOk ? (
          <BlockSpacer>
            <Text variant="body" color="primary">
              Health:{' '}
              <Text variant="label" color="success" as="span">
                {data?.status}
              </Text>
              {data?.time ? (
                <>
                  {' '}
                  at{' '}
                  <Text variant="code" color="steel" as="span">
                    {data.time}
                  </Text>
                </>
              ) : null}
            </Text>
          </BlockSpacer>
        ) : null}
      </PanelCard>

      <PanelCard
        title={
          <Text variant="h4" color="cream" as="span">
            Chart host (stub sample)
          </Text>
        }
        size="small"
      >
        <Text variant="bodySm" color="steel">
          Sample candles for {APP_NAME} Lightweight Charts integration (not live data).
        </Text>
        <BlockSpacer>
          <CandleChartHost
            data={[
              { time: 1_700_000_000, open: 100, high: 110, low: 95, close: 105 },
              { time: 1_700_000_300, open: 105, high: 112, low: 102, close: 108 },
              { time: 1_700_000_600, open: 108, high: 115, low: 107, close: 111 },
              { time: 1_700_000_900, open: 111, high: 111, low: 100, close: 102 },
              { time: 1_700_001_200, open: 102, high: 120, low: 101, close: 118 },
            ]}
          />
        </BlockSpacer>
      </PanelCard>
    </PageStack>
  );
}
