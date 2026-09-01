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
import {
  LIQ_CHART_RANGES,
  LIQ_LEVELS_POLL_MS,
  LiquidationBarChart,
} from '@/components/organisms/LiquidationBarChart';
import {
  LIQ_CASCADE_POLL_MS,
  LiquidationCascade,
} from '@/components/organisms/LiquidationCascade';
import {
  LIQ_HUNT_POLL_MS,
  LiquidationHunt,
} from '@/components/organisms/LiquidationHunt';
import {
  LIQ_MAX_PAIN_POLL_MS,
  LiquidationMaxPain,
} from '@/components/organisms/LiquidationMaxPain';
import {
  huntWeightQueryParams,
  parseHuntWeightDraft,
} from '@/components/organisms/LiquidationHunt/helpers';
import { LiquidationFeedHealth } from '@/components/organisms/LiquidationFeedHealth';
import { LiquidationTreemap } from '@/components/organisms/LiquidationTreemap';
import { LiquidationWindowCards } from '@/components/organisms/LiquidationWindowCards';
import {
  rtkErrorMessage,
  useGetMarketLiquidationCascadeQuery,
  useGetMarketLiquidationCascadeScanQuery,
  useGetMarketLiquidationHuntQuery,
  useGetMarketLiquidationMaxPainQuery,
  useGetMarketLiquidationHuntHeatmapQuery,
  useGetMarketLiquidationLevelsQuery,
  useGetMarketLiquidationOverviewQuery,
  useGetTicker24hQuery,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import { DEFAULT_LIQ_SYMBOL, LIQ_OVERVIEW_POLL_MS } from './LiquidationsPage.constants';
import {
  parseLiqCascadeSymbol,
  parseLiqChartRange,
  parseLiqChartSymbol,
  parseLiqExchange,
  parseLiqSymbol,
  parseLiqHuntPanel,
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
  const huntPanel = parseLiqHuntPanel(searchParams.get('panel'));
  const huntWeights = parseHuntWeightDraft((key) => searchParams.get(key));
  const windowId = parseLiqWindow(searchParams.get('window'));
  const exchange = parseLiqExchange(searchParams.get('exchange'));
  const symbol = parseLiqSymbol(searchParams.get('symbol'));
  const chartSymbol = parseLiqChartSymbol(searchParams.get('symbol'));
  const cascadeSymbol = parseLiqCascadeSymbol(searchParams.get('symbol'));
  const chartRange = parseLiqChartRange(searchParams.get('range'));
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
      pollingInterval:
        visible && (view === 'overview' || view === 'chart' || view === 'cascade')
          ? LIQ_OVERVIEW_POLL_MS
          : 0,
      refetchOnFocus: true,
    },
  );
  const overview = rtkCurrent(overviewQuery);
  const coins = overview?.coins ?? [];

  const levelsQuery = useGetMarketLiquidationLevelsQuery(
    { exchange, symbol: chartSymbol, range: chartRange },
    {
      skip: view !== 'chart',
      pollingInterval: visible && view === 'chart' ? LIQ_LEVELS_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const maxPainQuery = useGetMarketLiquidationMaxPainQuery(
    { symbol, exchange: exchange === 'all' ? 'all' : exchange },
    {
      skip: view !== 'max-pain' || !symbol,
      pollingInterval: visible && view === 'max-pain' ? LIQ_MAX_PAIN_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const huntQuery = useGetMarketLiquidationHuntQuery(
    {
      symbol,
      exchange: exchange === 'all' ? 'all' : exchange,
      ...(huntWeights ? huntWeightQueryParams(huntWeights) : {}),
    },
    {
      skip: view !== 'hunt' || !symbol,
      pollingInterval: visible && view === 'hunt' ? LIQ_HUNT_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const heatQuery = useGetMarketLiquidationHuntHeatmapQuery(
    { symbol, range: liqHeatRange, exchange: exchange === 'all' ? 'all' : exchange },
    {
      skip: view !== 'heatmap' || !symbol,
      pollingInterval: visible && view === 'heatmap' ? LIQ_HEATMAP_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const heatTickerExchange = liqHeatVenue === 'bybit' ? 'bybit' : liqHeatVenue === 'binance' ? 'binance' : null;
  const tickerQuery = useGetTicker24hQuery(
    { symbol, exchange: heatTickerExchange ?? 'binance' },
    { skip: view !== 'heatmap' || !symbol || heatTickerExchange == null },
  );

  const cascadeScanQuery = useGetMarketLiquidationCascadeScanQuery(
    { exchange },
    {
      skip: view !== 'cascade',
      pollingInterval: visible && view === 'cascade' ? LIQ_CASCADE_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const cascadeQuery = useGetMarketLiquidationCascadeQuery(
    { exchange, symbol: cascadeSymbol },
    {
      skip: view !== 'cascade' || cascadeSymbol === 'all',
      pollingInterval: visible && view === 'cascade' && cascadeSymbol !== 'all' ? LIQ_CASCADE_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const cascadeScan = rtkCurrent(cascadeScanQuery);
  const cascadeReport =
    cascadeSymbol === 'all' ? cascadeScan?.market : rtkCurrent(cascadeQuery);

  const symbolOptions = useMemo(() => {
    const fromCoins = coins
      .map((c) => c.symbol)
      .filter((s): s is string => Boolean(s));
    const extra = [symbol, cascadeSymbol].filter(
      (s): s is string => Boolean(s) && s !== 'all' && !fromCoins.includes(s),
    );
    if (extra.length) {
      return [...extra, ...fromCoins];
    }
    return fromCoins.length ? fromCoins : [DEFAULT_LIQ_SYMBOL];
  }, [coins, symbol, cascadeSymbol]);

  return (
    <PageStack>
      <PageHeader
        title={t('liquidations:title')}
        subtitle={t('liquidations:subtitle')}
      />

      <Toolbar>
        <Segmented
          value={view}
          onChange={(next) =>
            patchParams({
              view: next === 'overview' ? null : String(next),
            })
          }
          options={[
            { value: 'overview', label: t('liquidations:tabs.overview') },
            { value: 'chart', label: t('liquidations:tabs.chart') },
            { value: 'cascade', label: t('liquidations:tabs.cascade') },
            { value: 'max-pain', label: t('liquidations:tabs.maxPain') },
            { value: 'hunt', label: t('liquidations:tabs.hunt') },
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
        {view === 'hunt' || view === 'heatmap' || view === 'max-pain' ? (
          <Field>
            <Text variant="caption" color="secondary" id="liq-symbol-label">
              {t(
                view === 'hunt'
                  ? 'liquidations:hunt.symbol'
                  : view === 'max-pain'
                    ? 'liquidations:maxPain.symbol'
                    : 'liquidations:heatmap.symbol',
              )}
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
        ) : null}
        {view === 'cascade' ? (
          <Field>
            <Text variant="caption" color="secondary" id="liq-cascade-coin-label">
              {t('liquidations:cascade.coin')}
            </Text>
            <Select
              aria-labelledby="liq-cascade-coin-label"
              showSearch
              value={cascadeSymbol}
              style={{ minWidth: 168 }}
              options={[
                { value: 'all', label: t('liquidations:cascade.market') },
                ...symbolOptions.map((s) => ({
                  value: s,
                  label: s.replace(/USDT$|USDC$/i, '') || s,
                })),
              ]}
              onChange={(next) => patchParams({ symbol: next === 'all' ? 'all' : next })}
            />
          </Field>
        ) : null}
        {view === 'chart' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary" id="liq-coin-label">
                {t('liquidations:chart.coin')}
              </Text>
              <Select
                aria-labelledby="liq-coin-label"
                showSearch
                value={chartSymbol}
                style={{ minWidth: 160 }}
                options={[
                  { value: 'all', label: t('liquidations:chart.allCoins') },
                  ...symbolOptions.map((s) => ({ value: s, label: s.replace(/USDT$|USDC$/i, '') || s })),
                ]}
                onChange={(next) => patchParams({ symbol: next === DEFAULT_LIQ_SYMBOL ? null : next })}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary" id="liq-range-label">
                {t('liquidations:chart.range')}
              </Text>
              <Select
                aria-labelledby="liq-range-label"
                value={chartRange}
                style={{ minWidth: 88 }}
                options={LIQ_CHART_RANGES.map((r) => ({ value: r, label: r }))}
                onChange={(next) => patchParams({ range: next === '24h' ? null : next })}
              />
            </Field>
          </>
        ) : null}
      </Toolbar>

      <LiquidationFeedHealth feed={overview?.feed ?? rtkCurrent(levelsQuery)?.feed} />

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
              onOpen={(next) => patchParams({ view: 'chart', symbol: next })}
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
      ) : view === 'chart' ? (
        <LiquidationBarChart
          data={rtkCurrent(levelsQuery)}
          isLoading={rtkCurrentPending(levelsQuery)}
          isFetching={levelsQuery.isFetching}
          errorMessage={
            levelsQuery.isError
              ? rtkErrorMessage(levelsQuery.error, {
                  resource: t('liquidations:tabs.chart'),
                })
              : null
          }
        />
      ) : view === 'cascade' ? (
        <LiquidationCascade
          report={cascadeReport}
          hits={cascadeScan?.hits ?? []}
          isLoading={
            cascadeSymbol === 'all'
              ? rtkCurrentPending(cascadeScanQuery)
              : rtkCurrentPending(cascadeQuery)
          }
          isFetching={cascadeScanQuery.isFetching || cascadeQuery.isFetching}
          errorMessage={
            cascadeScanQuery.isError
              ? rtkErrorMessage(cascadeScanQuery.error, {
                  resource: t('liquidations:tabs.cascade'),
                })
              : cascadeQuery.isError
                ? rtkErrorMessage(cascadeQuery.error, {
                    resource: t('liquidations:tabs.cascade'),
                  })
                : null
          }
          onOpenCoin={(next) => patchParams({ symbol: next })}
        />
      ) : view === 'max-pain' ? (
        symbol ? (
          <LiquidationMaxPain
            data={rtkCurrent(maxPainQuery)}
            isLoading={rtkCurrentPending(maxPainQuery)}
            errorMessage={
              maxPainQuery.isError
                ? rtkErrorMessage(maxPainQuery.error, {
                    resource: t('liquidations:tabs.maxPain'),
                  })
                : null
            }
          />
        ) : (
          <Text variant="bodySm" color="secondary">
            {t('liquidations:maxPain.pick')}
          </Text>
        )
      ) : view === 'hunt' ? (
        symbol ? (
          <LiquidationHunt
            data={rtkCurrent(huntQuery)}
            isLoading={rtkCurrentPending(huntQuery)}
            isFetching={huntQuery.isFetching}
            panel={huntPanel}
            onPanelChange={(next) => patchParams({ panel: next === 'compare' ? null : next })}
            weightDraft={huntWeights ?? undefined}
            onApplyWeights={(draft) => {
              if (!draft) {
                patchParams({
                  weightProximity: null,
                  weightBook: null,
                  weightEfficiency: null,
                  weightTrend: null,
                  weightCrowding: null,
                  weightFlow: null,
                });
                return;
              }
              const q = huntWeightQueryParams(draft);
              patchParams({
                weightProximity: String(q.weightProximity),
                weightBook: String(q.weightBook),
                weightEfficiency: String(q.weightEfficiency),
                weightTrend: String(q.weightTrend),
                weightCrowding: String(q.weightCrowding),
                weightFlow: String(q.weightFlow),
              });
            }}
            errorMessage={
              huntQuery.isError
                ? rtkErrorMessage(huntQuery.error, {
                    resource: t('liquidations:tabs.hunt'),
                  })
                : null
            }
          />
        ) : (
          <Text variant="bodySm" color="secondary">
            {t('liquidations:hunt.pick')}
          </Text>
        )
      ) : (
        <HeatmapStack>
          {symbol ? (
            <LiquidationHeatmap
              data={rtkCurrent(heatQuery)}
              range={liqHeatRange}
              onRangeChange={setLiqHeatRange}
              venue={liqHeatVenue}
              onVenueChange={setLiqHeatVenue}
              side={liqHeatSide}
              onSideChange={setLiqHeatSide}
              lastPrice={
                liqHeatVenue === 'combined'
                  ? undefined
                  : Number(
                      rtkCurrent(heatQuery)?.[liqHeatVenue]?.lastPrice ||
                        rtkCurrent(tickerQuery)?.lastPrice,
                    ) || undefined
              }
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
