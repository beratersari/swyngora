import { useMemo, useState } from 'react';
import { Alert, Segmented, Select } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import {
  DEFAULT_LIQ_HEATMAP_RANGE,
  LIQ_HEATMAP_POLL_MS,
  LiquidationHeatmap,
  type LiqHeatRange,
  type LiqHeatSide,
  type LiqHeatVenue,
} from '@/components/organisms/LiquidationHeatmap';
import { LiquidationTreemap } from '@/components/organisms/LiquidationTreemap';
import { LiquidationWindowCards } from '@/components/organisms/LiquidationWindowCards';
import {
  rtkErrorMessage,
  useGetMarketLiquidationHuntHeatmapQuery,
  useGetMarketLiquidationOverviewQuery,
  useGetTicker24hQuery,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import { DEFAULT_LIQ_SYMBOL, LIQ_OVERVIEW_POLL_MS } from './LiquidationsPage.constants';
import {
  parseLiqExchange,
  parseLiqSymbol,
  parseLiqView,
  parseLiqWindow,
  type LiqPageExchange,
} from './LiquidationsPage.helpers';
import {
  CardsCol,
  Field,
  HeatmapStack,
  MapCol,
  OverviewLayout,
  PageStack,
  Toolbar,
} from './LiquidationsPage.styles';

export function LiquidationsPage() {
  const { t } = useTranslation(['liquidations', 'common']);
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const view = parseLiqView(searchParams.get('view'));
  const windowId = parseLiqWindow(searchParams.get('window'));
  const exchange = parseLiqExchange(searchParams.get('exchange'));
  const symbol = parseLiqSymbol(searchParams.get('symbol'));
  const [liqHeatRange, setLiqHeatRange] = useState<LiqHeatRange>(DEFAULT_LIQ_HEATMAP_RANGE);
  const [liqHeatVenue, setLiqHeatVenue] = useState<LiqHeatVenue>('combined');
  const [liqHeatSide, setLiqHeatSide] = useState<LiqHeatSide>('totals');

  const patchParams = (next: Record<string, string | null>) => {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        for (const [key, value] of Object.entries(next)) {
          if (value == null || value === '') p.delete(key);
          else p.set(key, value);
        }
        return p;
      },
      { replace: true },
    );
  };

  const overviewQuery = useGetMarketLiquidationOverviewQuery(
    { exchange, window: windowId, limit: 80 },
    {
      pollingInterval: visible && view === 'overview' ? LIQ_OVERVIEW_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const overview = rtkCurrent(overviewQuery);
  const coins = overview?.coins ?? [];

  const heatQuery = useGetMarketLiquidationHuntHeatmapQuery(
    { symbol, range: liqHeatRange, exchange: exchange === 'all' ? 'all' : exchange },
    {
      skip: view !== 'heatmap' || !symbol,
      pollingInterval: visible && view === 'heatmap' ? LIQ_HEATMAP_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const tickerQuery = useGetTicker24hQuery(
    { symbol, exchange: exchange === 'bybit' ? 'bybit' : 'binance' },
    { skip: view !== 'heatmap' || !symbol },
  );

  const symbolOptions = useMemo(() => {
    const fromCoins = coins
      .map((c) => c.symbol)
      .filter((s): s is string => Boolean(s));
    if (symbol && !fromCoins.includes(symbol)) {
      return [symbol, ...fromCoins];
    }
    return fromCoins.length ? fromCoins : [DEFAULT_LIQ_SYMBOL];
  }, [coins, symbol]);

  return (
    <PageStack>
      <PageHeader
        title={t('liquidations:title')}
        subtitle={t('liquidations:subtitle')}
      />

      <Toolbar>
        <Segmented
          value={view}
          onChange={(next) => patchParams({ view: next === 'heatmap' ? 'heatmap' : null })}
          options={[
            { value: 'overview', label: t('liquidations:tabs.overview') },
            { value: 'heatmap', label: t('liquidations:tabs.heatmap') },
          ]}
        />
        <Field>
          <Text variant="caption" color="secondary" id="liq-exchange-label">
            {t('liquidations:exchange')}
          </Text>
          <Select
            aria-labelledby="liq-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={[
              { value: 'all', label: t('liquidations:exchangeAll') },
              { value: 'binance', label: t('common:exchanges.binance') },
              { value: 'bybit', label: t('common:exchanges.bybit') },
            ]}
            onChange={(next: LiqPageExchange) =>
              patchParams({ exchange: next === 'all' ? null : next })
            }
          />
        </Field>
      </Toolbar>

      {overviewQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={rtkErrorMessage(overviewQuery.error, {
            resource: t('liquidations:title'),
          })}
        />
      ) : null}

      {view === 'overview' ? (
        <OverviewLayout>
          <MapCol>
            <LiquidationTreemap
              coins={coins}
              isLoading={rtkCurrentPending(overviewQuery)}
              onOpen={(next) => patchParams({ view: 'heatmap', symbol: next })}
            />
          </MapCol>
          <CardsCol>
            <LiquidationWindowCards
              windows={overview?.windows ?? []}
              selectedWindow={windowId}
              onSelect={(next) => patchParams({ window: next === '24h' ? null : next })}
              isLoading={rtkCurrentPending(overviewQuery)}
            />
          </CardsCol>
        </OverviewLayout>
      ) : (
        <HeatmapStack>
          <Field>
            <Text variant="caption" color="secondary" id="liq-symbol-label">
              {t('liquidations:heatmap.symbol')}
            </Text>
            <Select
              aria-labelledby="liq-symbol-label"
              showSearch
              value={symbol}
              style={{ minWidth: 160 }}
              options={symbolOptions.map((s) => ({ value: s, label: s }))}
              onChange={(next) => patchParams({ symbol: next })}
            />
          </Field>
          {symbol ? (
            <LiquidationHeatmap
              data={rtkCurrent(heatQuery)}
              range={liqHeatRange}
              onRangeChange={setLiqHeatRange}
              venue={liqHeatVenue}
              onVenueChange={setLiqHeatVenue}
              side={liqHeatSide}
              onSideChange={setLiqHeatSide}
              lastPrice={Number(rtkCurrent(tickerQuery)?.lastPrice)}
              isLoading={rtkCurrentPending(heatQuery)}
              isFetching={heatQuery.isFetching}
              errorMessage={
                heatQuery.isError
                  ? rtkErrorMessage(heatQuery.error, {
                      resource: t('liquidations:tabs.heatmap'),
                    })
                  : null
              }
            />
          ) : (
            <Text variant="bodySm" color="secondary">
              {t('liquidations:heatmap.pick')}
            </Text>
          )}
        </HeatmapStack>
      )}

      <Text variant="caption" color="secondary">
        {t('liquidations:disclaimer')}
      </Text>
    </PageStack>
  );
}
