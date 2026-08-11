import { useMemo, useState } from 'react';
import { Alert, Button, Tabs, message } from 'antd';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SignalsBacktestPanel } from '@/components/organisms/SignalsBacktestPanel';
import { SignalsHitsTable } from '@/components/organisms/SignalsHitsTable';
import { SignalsRuleForm } from '@/components/organisms/SignalsRuleForm';
import { SignalsRulesTable } from '@/components/organisms/SignalsRulesTable';
import { SignalsSetupGrid } from '@/components/organisms/SignalsSetupGrid';
import { SwingEngineGrid } from '@/components/organisms/SwingEngineGrid';
import {
  rtkErrorMessage,
  useCancelScannerBacktestMutation,
  useCreateScannerRuleMutation,
  useDeleteScannerRuleMutation,
  useGetWatchlistQuery,
  useListIntervalsQuery,
  useListScannerBacktestSignalsQuery,
  useListScannerBacktestsQuery,
  useListScannerResultsQuery,
  useListScannerRulesQuery,
  useStartScannerBacktestMutation,
  useListSwingSetupsQuery,
  type CreateScannerRuleArg,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { backtestRangeIso, buildSwingSetups, countHitsSince } from '@/libs/utils';
import {
  SIGNALS_BACKTEST_POLL_MS,
  SIGNALS_BACKTEST_RANGES,
  SIGNALS_INTERVAL_FALLBACK,
  SIGNALS_RESULT_LIMIT,
  SIGNALS_RESULTS_POLL_MS,
  SWING_STACK_INTERVAL,
  SWING_STACK_RULES,
} from './SignalsPage.constants';
import { DeskSplit, MetricCard, MetricStrip, PageStack, Section } from './SignalsPage.styles';

/**
 * Swing-signal desk: watchlist scanner rules, confluence setups, hits, backtests.
 * Informational only — not financial advice.
 */
export function SignalsPage() {
  const { t } = useTranslation(['signals', 'common', 'watchlist']);
  const navigate = useNavigate();
  const visible = useDocumentVisible();
  const [tab, setTab] = useState('setups');
  const [selectedBacktestId, setSelectedBacktestId] = useState<string | null>(null);

  const watchlist = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const intervalsQuery = useListIntervalsQuery({ exchange: 'binance' });
  const rulesQuery = useListScannerRulesQuery(undefined, { refetchOnFocus: true });
  const resultsQuery = useListScannerResultsQuery(
    { limit: SIGNALS_RESULT_LIMIT, offset: 0 },
    {
      pollingInterval: visible ? SIGNALS_RESULTS_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const engineQuery = useListSwingSetupsQuery(
    { limit: 25 },
    {
      pollingInterval: visible ? SIGNALS_RESULTS_POLL_MS * 2 : 0,
      refetchOnFocus: true,
      skip: !watchlist.isLoading && (watchlist.data?.items?.length ?? 0) === 0,
    },
  );
  const backtestsQuery = useListScannerBacktestsQuery(undefined, {
    pollingInterval: visible ? SIGNALS_BACKTEST_POLL_MS : 0,
    refetchOnFocus: true,
  });

  const [createRule, createState] = useCreateScannerRuleMutation();
  const [deleteRule, deleteState] = useDeleteScannerRuleMutation();
  const [startBacktest, startState] = useStartScannerBacktestMutation();
  const [cancelBacktest, cancelState] = useCancelScannerBacktestMutation();

  const selectedSignals = useListScannerBacktestSignalsQuery(
    { id: selectedBacktestId ?? '', limit: 100 },
    { skip: !selectedBacktestId },
  );

  const rules = rulesQuery.data?.rules ?? [];
  const results = resultsQuery.data?.results ?? [];
  const jobs = backtestsQuery.data?.backtests ?? [];
  const watchCount = watchlist.data?.items?.length ?? 0;

  const setups = useMemo(() => buildSwingSetups(results), [results]);
  const engineItems = engineQuery.data?.items ?? [];
  const engineAccepted = engineQuery.data?.accepted ?? 0;
  const hits24h = useMemo(() => countHitsSince(results, Date.now() - 24 * 60 * 60 * 1000), [results]);
  const activeJobs = jobs.filter((j) => j.status === 'pending' || j.status === 'running').length;

  const intervals = intervalsQuery.data?.intervals?.length
    ? intervalsQuery.data.intervals
    : [...SIGNALS_INTERVAL_FALLBACK];

  const selectedJob = jobs.find((j) => j.id === selectedBacktestId) ?? null;

  const rangeOptions = SIGNALS_BACKTEST_RANGES.map((r) => ({
    value: r.key,
    label: t(`signals:lab.ranges.${r.key}`),
  }));

  const openChart = (exchange: string, symbol: string) => {
    navigate(`/markets/${encodeURIComponent(exchange)}/${encodeURIComponent(symbol)}`);
  };

  const addStack = async () => {
    const interval = intervals.includes(SWING_STACK_INTERVAL) ? SWING_STACK_INTERVAL : (intervals[0] ?? '1h');
    try {
      for (const rule of SWING_STACK_RULES) {
        await createRule({ ...rule, interval }).unwrap();
      }
      void message.success(t('signals:rules.stackSuccess'));
      void rulesQuery.refetch();
    } catch (err) {
      void message.error(rtkErrorMessage(err, { resource: t('signals:resource') }));
    }
  };

  const listError = rulesQuery.isError || resultsQuery.isError;

  return (
    <PageStack>
      <PageHeader
        eyebrow={t('signals:eyebrow')}
        title={t('signals:title')}
        subtitle={t('signals:subtitle')}
        extra={
          <Button onClick={() => void addStack()} loading={createState.isLoading} disabled={createState.isLoading}>
            {t('signals:rules.addStack')}
          </Button>
        }
      />

      <Alert type="info" showIcon message={t('signals:disclaimer')} />

      {watchCount === 0 && !watchlist.isLoading ? (
        <Alert
          type="warning"
          showIcon
          message={t('signals:watchlistEmptyTitle')}
          description={
            <>
              {t('signals:watchlistEmptyBody')}{' '}
              <Link to="/watchlist">{t('watchlist:title')}</Link>
            </>
          }
        />
      ) : null}

      {listError ? (
        <Alert
          type="error"
          showIcon
          message={t('signals:loadFailed')}
          description={rtkErrorMessage(rulesQuery.error ?? resultsQuery.error, {
            resource: t('signals:resource'),
          })}
          action={
            <Button
              size="small"
              onClick={() => {
                void rulesQuery.refetch();
                void resultsQuery.refetch();
              }}
            >
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      <MetricStrip>
        <MetricCard>
          <Text variant="caption" color="secondary">
            {t('signals:metrics.rules')}
          </Text>
          <Text variant="h3" color="primary" mono>
            {rules.length}
          </Text>
        </MetricCard>
        <MetricCard>
          <Text variant="caption" color="secondary">
            {t('signals:metrics.hits24h')}
          </Text>
          <Text variant="h3" color="primary" mono>
            {hits24h}
          </Text>
        </MetricCard>
        <MetricCard>
          <Text variant="caption" color="secondary">
            {t('signals:metrics.setups')}
          </Text>
          <Text variant="h3" color="primary" mono>
            {engineAccepted || setups.length}
          </Text>
        </MetricCard>
        <MetricCard>
          <Text variant="caption" color="secondary">
            {t('signals:metrics.backtests')}
          </Text>
          <Text variant="h3" color="primary" mono>
            {activeJobs}
          </Text>
        </MetricCard>
      </MetricStrip>

      <Tabs
        activeKey={tab}
        onChange={setTab}
        items={[
          {
            key: 'setups',
            label: t('signals:tabs.setups'),
            children: (
              <Section>
                <Text variant="caption" color="secondary">
                  {t('signals:engine.hint')}
                </Text>
                <SwingEngineGrid
                  items={engineItems}
                  loading={engineQuery.isFetching || engineQuery.isLoading}
                  onOpen={openChart}
                />
                <Text variant="caption" color="secondary">
                  {t('signals:setups.hint')}
                </Text>
                <SignalsSetupGrid
                  setups={setups}
                  loading={resultsQuery.isLoading}
                  onOpen={openChart}
                />
              </Section>
            ),
          },
          {
            key: 'hits',
            label: t('signals:tabs.hits'),
            children: (
              <SignalsHitsTable
                items={results}
                loading={resultsQuery.isLoading}
                onOpen={openChart}
              />
            ),
          },
          {
            key: 'rules',
            label: t('signals:tabs.rules'),
            children: (
              <DeskSplit>
                <SignalsRuleForm
                  intervals={intervals}
                  defaultInterval={intervals.includes('4h') ? '4h' : intervals[0]}
                  isSubmitting={createState.isLoading}
                  submitError={createState.isError ? createState.error : undefined}
                  onSubmit={async (values: CreateScannerRuleArg) => {
                    await createRule(values).unwrap();
                    void message.success(t('signals:rules.createSuccess'));
                    void rulesQuery.refetch();
                  }}
                />
                <div>
                  <Text variant="h4" color="primary">
                    {t('signals:rules.listTitle')}
                  </Text>
                  <SignalsRulesTable
                    items={rules}
                    loading={rulesQuery.isLoading}
                    deleteLoading={deleteState.isLoading}
                    onDelete={(id) => {
                      void deleteRule({ id })
                        .unwrap()
                        .then(() => {
                          void message.success(t('signals:rules.deleteSuccess'));
                          void rulesQuery.refetch();
                          void resultsQuery.refetch();
                        })
                        .catch((err: unknown) => {
                          void message.error(
                            rtkErrorMessage(err, { resource: t('signals:resource') }),
                          );
                        });
                    }}
                  />
                </div>
              </DeskSplit>
            ),
          },
          {
            key: 'lab',
            label: t('signals:tabs.lab'),
            children: (
              <SignalsBacktestPanel
                rules={rules}
                jobs={jobs}
                signals={selectedSignals.data?.signals ?? []}
                rangeOptions={rangeOptions}
                selectedId={selectedBacktestId}
                selectedJob={selectedJob}
                loading={backtestsQuery.isLoading}
                signalsLoading={selectedSignals.isFetching}
                startLoading={startState.isLoading}
                cancelLoading={cancelState.isLoading}
                startError={startState.isError ? startState.error : undefined}
                onStart={({ ruleId, symbol, exchange, rangeKey }) => {
                  const days =
                    SIGNALS_BACKTEST_RANGES.find((r) => r.key === rangeKey)?.days ?? 90;
                  const { rangeStart, rangeEnd } = backtestRangeIso(days);
                  void startBacktest({ ruleId, symbol, exchange, rangeStart, rangeEnd })
                    .unwrap()
                    .then((job) => {
                      void message.success(t('signals:lab.startSuccess'));
                      setSelectedBacktestId(job.id);
                      void backtestsQuery.refetch();
                    });
                }}
                onSelect={(id) => setSelectedBacktestId(id || null)}
                onCancel={(id) => {
                  void cancelBacktest({ id })
                    .unwrap()
                    .then(() => void backtestsQuery.refetch())
                    .catch((err: unknown) => {
                      void message.error(rtkErrorMessage(err, { resource: t('signals:resource') }));
                    });
                }}
              />
            ),
          },
        ]}
      />
    </PageStack>
  );
}
