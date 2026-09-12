import { useState } from 'react';
import { Button, InputNumber, Segmented, Table, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { FundingArbScanTable } from '@/components/organisms/FundingArbScanTable';
import { PriceDiffScanTable } from '@/components/organisms/PriceDiffScanTable';
import {
  rtkErrorMessage,
  useCreateFundingArbWatchMutation,
  useCreatePriceDiffWatchMutation,
  useDeleteFundingArbWatchMutation,
  useDeletePriceDiffWatchMutation,
  useLazyScanMarketFundingArbQuery,
  useLazyScanPriceDiffQuotesQuery,
  useListFundingArbWatchesQuery,
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
  const [scan, scanState] = useLazyScanPriceDiffQuotesQuery();
  const watches = useListPriceDiffWatchesQuery();
  const opps = useListPriceDiffOpportunitiesQuery({ status: 'open' });
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
        <Table
          size="small"
          rowKey={(r) => r.id ?? `${r.symbol}-${r.buyExchange}`}
          pagination={false}
          dataSource={opps.data?.opportunities ?? []}
          locale={{ emptyText: t('arb:price.noOpps') }}
          columns={[
            { title: t('arb:price.symbol'), dataIndex: 'symbol' },
            {
              title: t('arb:price.route'),
              render: (_, r) => `${r.buyExchange ?? ''} → ${r.sellExchange ?? ''}`,
            },
            { title: t('arb:price.netPct'), dataIndex: 'netDiffPct' },
            { title: t('arb:price.status'), dataIndex: 'status' },
          ]}
        />
      </Section>
    </>
  );
}

function FundingPane() {
  const { t } = useTranslation(['arb', 'common']);
  const [notional, setNotional] = useState(10000);
  const [scan, scanState] = useLazyScanMarketFundingArbQuery();
  const watches = useListFundingArbWatchesQuery();
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
