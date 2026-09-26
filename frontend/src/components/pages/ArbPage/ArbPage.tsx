import { useState } from 'react';
import { Button, Input, InputNumber, Segmented, Table, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { defaultFundingHistoryRange, FundingArbHistoryTable } from '@/components/organisms/FundingArbHistoryTable';
import { FundingArbScanTable } from '@/components/organisms/FundingArbScanTable';
import { PriceDiffOpportunityTable, type PriceDiffOppStatusFilter } from '@/components/organisms/PriceDiffOpportunityTable';
import { PriceDiffScanTable } from '@/components/organisms/PriceDiffScanTable';
import {
  rtkErrorMessage,
  useCreateFundingArbWatchMutation,
  useCreatePriceDiffWatchMutation,
  useDeleteFundingArbWatchMutation,
  useDeletePriceDiffWatchMutation,
  useLazyGetMarketFundingArbHistoryQuery,
  useLazyScanMarketFundingArbQuery,
  useLazyScanPriceDiffQuotesQuery,
  useListFundingArbWatchesQuery,
  useListFundingArbSignalsQuery,
  useGetPriceDiffOpportunityQuery,
  useListPriceDiffOpportunitiesQuery,
  useListPriceDiffWatchesQuery,
} from '@/libs/api';
import { Field, PageStack, Section, Toolbar } from './ArbPage.styles';

/**
 * Cross-exchange price-gap and funding-arb desk.
 * Informational only — not financial advice.
 */
export function ArbPage() {
  const { t } = useTranslation(['arb', 'common']);
  const [view, setView] = useState<'price' | 'funding'>('price');

  return (
    <PageStack>
      <PageHeader title={t('arb:title')} subtitle={t('arb:subtitle')} />
      <Segmented
        value={view}
        options={[
          { label: t('arb:tabs.price'), value: 'price' },
          { label: t('arb:tabs.funding'), value: 'funding' },
        ]}
        onChange={(v) => setView(v as 'price' | 'funding')}
      />
      {view === 'price' ? <PricePane /> : <FundingPane />}
    </PageStack>
  );
}

function PricePane() {
  const { t } = useTranslation(['arb', 'common']);
  const [symbol, setSymbol] = useState('BTCUSDT');
  const [notional, setNotional] = useState(10000);
  const [oppStatus, setOppStatus] = useState<PriceDiffOppStatusFilter>('all');
  const [expandedOpp, setExpandedOpp] = useState<string | undefined>();
  const [scan, scanState] = useLazyScanPriceDiffQuotesQuery();
  const watches = useListPriceDiffWatchesQuery();
  const opps = useListPriceDiffOpportunitiesQuery({ status: oppStatus });
  const detail = useGetPriceDiffOpportunityQuery(expandedOpp ?? '', { skip: !expandedOpp });
  const [createWatch, createState] = useCreatePriceDiffWatchMutation();
  const [delWatch] = useDeletePriceDiffWatchMutation();

  return (
    <>
      <Section>
        <Text variant="bodySm">{t('arb:price.hint')}</Text>
        <Toolbar>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:price.symbol')}
            </Text>
            <SymbolSuggest exchange="binance" value={symbol} onChange={setSymbol} />
          </Field>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:price.notional')}
            </Text>
            <InputNumber min={100} step={100} value={notional} onChange={(v) => setNotional(Number(v) || 0)} />
          </Field>
          <Button
            type="primary"
            loading={scanState.isFetching}
            onClick={() => {
              void scan({ symbol, notional }).unwrap().catch((err) => {
                message.error(rtkErrorMessage(err, { resource: t('arb:price.resource') }));
              });
            }}
          >
            {t('arb:price.scan')}
          </Button>
        </Toolbar>
        {scanState.data?.bestRoute ? (
          <Text variant="caption" color="secondary">
            {t('arb:price.best', {
              route: `${scanState.data.bestRoute.buyExchange} → ${scanState.data.bestRoute.sellExchange}`,
              profit: scanState.data.bestRoute.profitAfterFees ?? '—',
            })}
          </Text>
        ) : null}
        <PriceDiffScanTable scan={scanState.data} isLoading={scanState.isFetching} />
      </Section>

      <Section>
        <Text variant="label">{t('arb:price.follow')}</Text>
        <Toolbar>
          <Button
            loading={createState.isLoading}
            onClick={() => {
              void createWatch({ symbol, notional, minProfit: 10 })
                .unwrap()
                .then(() => message.success(t('arb:price.watchCreated')))
                .catch((err) => message.error(rtkErrorMessage(err, { resource: t('arb:price.resource') })));
            }}
          >
            {t('arb:price.watch')}
          </Button>
        </Toolbar>
        <Table
          size="small"
          rowKey={(r) => r.id ?? r.symbol ?? ''}
          pagination={false}
          dataSource={watches.data?.watches ?? []}
          locale={{ emptyText: t('arb:price.noWatches') }}
          columns={[
            { title: t('arb:price.symbol'), dataIndex: 'symbol' },
            { title: t('arb:price.notional'), dataIndex: 'notional' },
            { title: t('arb:price.status'), dataIndex: 'status' },
            {
              title: t('arb:price.remove'),
              render: (_, r) => (
                <Button type="link" danger onClick={() => r.id && void delWatch(r.id)}>
                  {t('arb:price.remove')}
                </Button>
              ),
            },
          ]}
        />
      </Section>

      <Section>
        <Text variant="label">{t('arb:price.openOpps')}</Text>
        <PriceDiffOpportunityTable
          opportunities={opps.data?.opportunities ?? []}
          isLoading={opps.isFetching}
          status={oppStatus}
          onStatusChange={setOppStatus}
          expandedId={expandedOpp}
          onExpand={setExpandedOpp}
          detail={detail.data}
          detailLoading={detail.isFetching}
        />
      </Section>
    </>
  );
}

function FundingPane() {
  const { t } = useTranslation(['arb', 'common']);
  const [notional, setNotional] = useState(10000);
  const [historySymbol, setHistorySymbol] = useState('BTCUSDT');
  const [historyRange, setHistoryRange] = useState(defaultFundingHistoryRange);
  const [scan, scanState] = useLazyScanMarketFundingArbQuery();
  const [loadHistory, historyState] = useLazyGetMarketFundingArbHistoryQuery();
  const watches = useListFundingArbWatchesQuery();
  const signals = useListFundingArbSignalsQuery({ status: 'all', limit: 20 });
  const [createWatch, createState] = useCreateFundingArbWatchMutation();
  const [delWatch] = useDeleteFundingArbWatchMutation();

  return (
    <>
      <Section>
        <Text variant="bodySm">{t('arb:funding.hint')}</Text>
        <Toolbar>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:funding.notional')}
            </Text>
            <InputNumber min={100} step={100} value={notional} onChange={(v) => setNotional(Number(v) || 0)} />
          </Field>
          <Button
            type="primary"
            loading={scanState.isFetching}
            onClick={() => {
              void scan({ notional, limit: 15 }).unwrap().catch((err) => {
                message.error(rtkErrorMessage(err, { resource: t('arb:funding.resource') }));
              });
            }}
          >
            {t('arb:funding.scan')}
          </Button>
          <Button
            loading={createState.isLoading}
            onClick={() => {
              void createWatch({ notional, minProfit: 5 })
                .unwrap()
                .then(() => message.success(t('arb:funding.watchCreated')))
                .catch((err) => message.error(rtkErrorMessage(err, { resource: t('arb:funding.resource') })));
            }}
          >
            {t('arb:funding.followScan')}
          </Button>
        </Toolbar>
        <FundingArbScanTable scan={scanState.data} isLoading={scanState.isFetching} />
      </Section>
      <Section>
        <Text variant="label">{t('arb:funding.history')}</Text>
        <Text variant="bodySm">{t('arb:funding.historyHint')}</Text>
        <Toolbar>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:funding.symbol')}
            </Text>
            <SymbolSuggest exchange="binance" value={historySymbol} onChange={setHistorySymbol} />
          </Field>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:funding.historyFrom')}
            </Text>
            <Input
              value={historyRange.from}
              onChange={(e) => setHistoryRange((prev) => ({ ...prev, from: e.target.value }))}
            />
          </Field>
          <Field>
            <Text variant="caption" color="secondary">
              {t('arb:funding.historyTo')}
            </Text>
            <Input
              value={historyRange.to}
              onChange={(e) => setHistoryRange((prev) => ({ ...prev, to: e.target.value }))}
            />
          </Field>
          <Button
            loading={historyState.isFetching}
            onClick={() => {
              void loadHistory({
                symbol: historySymbol,
                from: historyRange.from,
                to: historyRange.to,
                notional,
              })
                .unwrap()
                .catch((err) => {
                  message.error(rtkErrorMessage(err, { resource: t('arb:funding.resource') }));
                });
            }}
          >
            {t('arb:funding.historyLookup')}
          </Button>
        </Toolbar>
        <FundingArbHistoryTable history={historyState.data} isLoading={historyState.isFetching} />
      </Section>
      <Section>
        <Text variant="label">{t('arb:funding.signals')}</Text>
        <Text variant="caption" color="secondary">
          {t('arb:funding.signalsHint')}
        </Text>
        <Table
          size="small"
          rowKey={(r) => r.id ?? `${r.symbol}-${r.openedAt}`}
          pagination={false}
          dataSource={signals.data?.signals ?? []}
          locale={{ emptyText: t('arb:funding.noSignals') }}
          columns={[
            { title: t('arb:funding.symbol'), dataIndex: 'symbol' },
            {
              title: t('arb:funding.sides'),
              render: (_, r) => `${r.longExchange ?? ''} / ${r.shortExchange ?? ''}`,
            },
            {
              title: t('arb:funding.expected'),
              render: (_, r) => r.review?.expectedNet ?? r.expected?.expectedNet ?? r.netAfterFees ?? '—',
            },
            {
              title: t('arb:funding.actual'),
              render: (_, r) =>
                r.review?.complete ? (r.review.actualNet ?? '—') : t('arb:funding.pending'),
            },
            {
              title: t('arb:funding.delta'),
              render: (_, r) => (r.review?.complete ? (r.review.deltaNet ?? '—') : '—'),
            },
            {
              title: t('arb:funding.result'),
              render: (_, r) => {
                if (!r.review?.complete) return t('arb:funding.pending');
                return r.review.stillWorthIt ? t('arb:funding.stillGood') : t('arb:funding.notGood');
              },
            },
          ]}
        />
      </Section>
      <Section>
        <Text variant="label">{t('arb:funding.follows')}</Text>
        <Table
          size="small"
          rowKey={(r) => r.id ?? r.symbol ?? 'scan'}
          pagination={false}
          dataSource={watches.data?.watches ?? []}
          locale={{ emptyText: t('arb:funding.noWatches') }}
          columns={[
            {
              title: t('arb:funding.scope'),
              render: (_, r) => r.symbol || r.scope || 'scan',
            },
            { title: t('arb:funding.notional'), dataIndex: 'notional' },
            {
              title: t('arb:funding.remove'),
              render: (_, r) => (
                <Button type="link" danger onClick={() => r.id && void delWatch(r.id)}>
                  {t('arb:funding.remove')}
                </Button>
              ),
            },
          ]}
        />
      </Section>
    </>
  );
}
