import { Fragment, useMemo, useState } from 'react';
import { Alert, Button, Drawer, Empty, InputNumber, Progress, Select } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import {
  rtkErrorMessage,
  type MarketExchange,
  type ScannerBacktest,
  type ScannerBacktestHorizonStats,
  type ScannerBacktestBook,
  type ScannerBacktestMarketPerformance,
  type ScannerBacktestPerformance,
  type ScannerBacktestSignal,
  type ScannerBacktestTrade,
} from '@/libs/api';
import {
  changeTone,
  describeRule,
  formatChangePercent,
  formatExactDateTime,
  formatPrice,
  formatSymbolDisplay,
} from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import { formatLeverage, hasBacktestHitDetails, isWatchlistBacktest, tradePnlTone, watchlistMarketCount } from './helpers';
import {
  FactorLine,
  FactorWhy,
  Field,
  FieldRow,
  PerfGrid,
  PerfTable,
  SignalDrawerBody,
  Stack,
  TradePlan,
  TradeStats,
} from './SignalsBacktestPanel.styles';
import type { SignalsBacktestPanelProps } from './SignalsBacktestPanel.types';

function horizonLabel(days: number): string {
  if (days === 1) return '1d';
  if (days === 5) return '5d';
  return '20d';
}

function PerformanceSummary({ perf }: { perf: ScannerBacktestPerformance }) {
  const { t } = useTranslation(['signals']);
  const rows: { key: string; label: string; render: (h: ScannerBacktestHorizonStats) => string }[] = [
    {
      key: 'avg',
      label: t('signals:lab.avgReturn'),
      render: (h) => (h.average == null ? '—' : formatChangePercent(h.average)),
    },
    {
      key: 'med',
      label: t('signals:lab.medianReturn'),
      render: (h) => (h.median == null ? '—' : formatChangePercent(h.median)),
    },
    {
      key: 'win',
      label: t('signals:lab.winRate'),
      render: (h) => (h.winRate == null ? '—' : `${h.winRate}%`),
    },
    {
      key: 'n',
      label: t('signals:lab.sampleCount'),
      render: (h) => String(h.sampleCount),
    },
  ];
  const horizons = [perf.return1d, perf.return5d, perf.return20d];
  return (
    <PerfTable>
      <Text variant="label" color="primary">
        {t('signals:lab.performance')}
      </Text>
      <PerfGrid>
        <Text variant="caption" color="secondary">
          {' '}
        </Text>
        {horizons.map((h) => (
          <Text key={h.days} variant="caption" color="secondary" mono>
            {horizonLabel(h.days)}
          </Text>
        ))}
        {rows.map((row) => (
          <Fragment key={row.key}>
            <Text variant="caption" color="secondary">
              {row.label}
            </Text>
            {horizons.map((h) => (
              <Text
                key={`${row.key}-${h.days}`}
                variant="caption"
                mono
                color={
                  row.key === 'n'
                    ? 'secondary'
                    : row.key === 'win'
                      ? 'primary'
                      : changeTone(row.key === 'med' ? h.median : h.average)
                }
              >
                {row.render(h)}
              </Text>
            ))}
          </Fragment>
        ))}
      </PerfGrid>
      {perf.book ? <BookSummary book={perf.book} /> : null}
      {perf.trades && (perf.trades.sampleCount ?? 0) > 0 ? (
        <>
          <Text variant="label" color="primary">
            {t('signals:lab.trades')}
          </Text>
          <TradeStats>
            <Text variant="caption" color="secondary">
              {t('signals:lab.tpHits')}: {perf.trades.tpHits}
            </Text>
            <Text variant="caption" color="secondary">
              {t('signals:lab.slHits')}: {perf.trades.slHits}
            </Text>
            <Text variant="caption" color="secondary">
              {t('signals:lab.openTrades')}: {perf.trades.openCount}
            </Text>
            <Text variant="caption" color="primary" mono>
              {t('signals:lab.tradeWinRate')}: {perf.trades.winRate == null ? '—' : `${perf.trades.winRate}%`}
            </Text>
            <Text variant="caption" color={changeTone(perf.trades.average)} mono>
              {t('signals:lab.avgPnl')}: {perf.trades.average == null ? '—' : formatChangePercent(perf.trades.average)}
            </Text>
            <Text variant="caption" color={changeTone(perf.trades.median)} mono>
              {t('signals:lab.medianPnl')}: {perf.trades.median == null ? '—' : formatChangePercent(perf.trades.median)}
            </Text>
          </TradeStats>
        </>
      ) : null}
    </PerfTable>
  );
}

function formatBalance(n: number | undefined): string {
  if (n == null || !Number.isFinite(n)) return '—';
  return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function BookSummary({ book }: { book: ScannerBacktestBook }) {
  const { t } = useTranslation(['signals']);
  const rangeBal = book.rangeEndBalance ?? book.endingBalance;
  const afterBal = book.afterRangeBalance ?? book.endingBalance;
  const rangePnl = rangeBal - book.startingBalance;
  const afterPnl = afterBal - book.startingBalance;
  return (
    <>
      <Text variant="label" color="primary">
        {t('signals:lab.book')}
      </Text>
      <TradeStats>
        <Text variant="caption" color="secondary" mono>
          {t('signals:lab.bookStart')}: {formatBalance(book.startingBalance)}
        </Text>
        <Text variant="caption" color="secondary">
          {t('signals:lab.riskPct')}: {book.riskPct ?? book.stakePct ?? '—'}%
          {book.maxOpenRiskPct != null ? ` / ${t('signals:lab.maxOpenRisk')}: ${book.maxOpenRiskPct}%` : ''}
        </Text>
        <Text variant="caption" color={changeTone(rangePnl)} mono>
          {t('signals:lab.bookRangeEnd')}: {formatBalance(rangeBal)}
        </Text>
        <Text variant="caption" color={changeTone(rangePnl)} mono>
          {t('signals:lab.bookRangePnl')}: {formatChangePercent((rangePnl / book.startingBalance) * 100)}
        </Text>
        {(book.closedAfterRange ?? 0) > 0 || (book.openAtRangeEnd ?? 0) > 0 ? (
          <>
            <Text variant="caption" color="secondary">
              {t('signals:lab.bookLaterClosed')}: {book.closedAfterRange ?? 0}
            </Text>
            <Text variant="caption" color={changeTone(afterPnl)} mono>
              {t('signals:lab.bookAfter')}: {formatBalance(afterBal)}
            </Text>
            <Text variant="caption" color={changeTone(afterPnl)} mono>
              {t('signals:lab.bookAfterPnl')}: {formatChangePercent((afterPnl / book.startingBalance) * 100)}
            </Text>
          </>
        ) : null}
        <Text variant="caption" color="secondary">
          {t('signals:lab.bookTaken')}: {book.tradesTaken ?? 0}
        </Text>
        <Text variant="caption" color="secondary">
          {t('signals:lab.bookSkipped')}: {book.tradesSkipped ?? 0}
          {(book.skippedNoRisk ?? 0) > 0 ? ` (${t('signals:lab.bookSkippedRisk')}: ${book.skippedNoRisk})` : ''}
        </Text>
      </TradeStats>
    </>
  );
}

function marketStatusLabel(t: (key: string) => string, row: ScannerBacktestMarketPerformance): string {
  if (row.status === 'error') return t('signals:lab.marketError');
  if (row.status === 'no_data') return t('signals:lab.marketNoData');
  if ((row.signalCount ?? 0) === 0) return t('signals:lab.marketNone');
  return t('signals:lab.marketOk');
}

function outcomeLabel(t: (key: string) => string, outcome: string | undefined): string {
  if (outcome === 'tp') return t('signals:lab.outcomeTp');
  if (outcome === 'sl') return t('signals:lab.outcomeSl');
  if (outcome === 'open') return t('signals:lab.outcomeOpen');
  return '—';
}

function sideLabel(t: (key: string) => string, side: string | undefined): string {
  if (side === 'short') return t('signals:lab.short');
  if (side === 'long') return t('signals:lab.long');
  return '—';
}

function TradeDetail({ trade }: { trade: ScannerBacktestTrade }) {
  const { t } = useTranslation(['signals']);
  return (
    <TradePlan>
      <Text variant="caption" color="secondary">
        {t('signals:lab.side')}: {sideLabel(t, trade.side)}
      </Text>
      <Text variant="caption" color="secondary" mono>
        {t('signals:lab.leverage')}: {formatLeverage(trade.leverage)}
      </Text>
      <Text variant="caption" color="secondary" mono>
        {t('signals:lab.entry')}: {formatPrice(trade.entry)}
      </Text>
      <Text variant="caption" color="secondary" mono>
        {t('signals:lab.stop')}: {formatPrice(trade.stopLoss)}
      </Text>
      <Text variant="caption" color="secondary" mono>
        {t('signals:lab.target')}: {formatPrice(trade.takeProfit)}
      </Text>
      <Text variant="caption" color="primary">
        {t('signals:lab.outcome')}: {outcomeLabel(t, trade.outcome)}
      </Text>
      <Text variant="caption" color={tradePnlTone(trade)} mono>
        {t('signals:lab.pnl')}: {formatChangePercent(trade.pnlPct)}
      </Text>
      {trade.barsHeld != null ? (
        <Text variant="caption" color="secondary" mono>
          {t('signals:lab.barsHeld')}: {trade.barsHeld}
        </Text>
      ) : null}
    </TradePlan>
  );
}

function statusVariant(status: string): 'live' | 'paused' | 'status' | 'delist' {
  if (status === 'completed') return 'live';
  if (status === 'running' || status === 'pending') return 'status';
  if (status === 'failed') return 'delist';
  return 'paused';
}

export function SignalsBacktestPanel({
  rules,
  profiles,
  jobs,
  signals,
  performance,
  rangeOptions,
  selectedId,
  selectedJob,
  loading,
  signalsLoading,
  startLoading,
  cancelLoading,
  startError,
  onStart,
  onSelect,
  onCancel,
}: SignalsBacktestPanelProps) {
  const { t, i18n } = useTranslation(['signals', 'common']);
  const [source, setSource] = useState<'rule' | 'profile'>('rule');
  const [ruleId, setRuleId] = useState(rules[0]?.id ?? '');
  const [profileId, setProfileId] = useState(profiles.find((p) => p.isDefault)?.id ?? 'built_in');
  const [exchange, setExchange] = useState<string>('binance');
  const [symbol, setSymbol] = useState('BTCUSDT');
  const [scope, setScope] = useState<'symbol' | 'watchlist'>('symbol');
  const [startingBalance, setStartingBalance] = useState(10000);
  const [riskPct, setRiskPct] = useState(1);
  const [maxOpenRiskPct, setMaxOpenRiskPct] = useState(3);
  const [rangeKey, setRangeKey] = useState(rangeOptions[1]?.value ?? rangeOptions[0]?.value ?? '90d');

  const ruleById = useMemo(() => new Map(rules.map((r) => [r.id, r])), [rules]);
  const watchlistRun = source === 'profile' && scope === 'watchlist';
  const canStart =
    !startLoading &&
    (source === 'rule' ? Boolean(ruleId) && Boolean(symbol.trim()) : Boolean(profileId) && (watchlistRun || Boolean(symbol.trim())));

  const jobColumns: ColumnsType<ScannerBacktest> = [
    {
      title: t('signals:symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Text variant="label" mono>
          {isWatchlistBacktest(row)
            ? t('signals:lab.watchlistCoins', { count: watchlistMarketCount(row) })
            : formatSymbolDisplay(row.symbol)}
        </Text>
      ),
    },
    {
      title: t('signals:exchange'),
      dataIndex: 'exchange',
      render: (ex: string, row) =>
        isWatchlistBacktest(row) ? (
          <Text variant="caption" color="secondary">
            —
          </Text>
        ) : (
          <BrandTag variant="exchange">{ex}</BrandTag>
        ),
    },
    {
      title: t('signals:lab.source'),
      key: 'rule',
      render: (_, row) => {
        if (row.kind === 'custom_profile') {
          return <Text variant="caption">{row.sourceName || t('signals:custom.builtIn')}</Text>;
        }
        const rule = ruleById.get(row.ruleId);
        return <Text variant="caption">{rule ? describeRule(rule) : row.ruleId}</Text>;
      },
    },
    {
      title: t('signals:lab.status'),
      dataIndex: 'status',
      render: (s: string) => <BrandTag variant={statusVariant(s)}>{s}</BrandTag>,
    },
    {
      title: t('signals:lab.progress'),
      key: 'progress',
      render: (_, row) => (
        <Progress
          percent={Math.round(row.progressPct || 0)}
          size="small"
          status={row.status === 'failed' ? 'exception' : row.status === 'completed' ? 'success' : 'active'}
        />
      ),
    },
    {
      title: t('signals:lab.signals'),
      dataIndex: 'signalCount',
    },
    {
      title: t('signals:lab.skipped'),
      key: 'skipped',
      render: (_, row) =>
        row.kind === 'custom_profile' && (row.skippedBars ?? 0) > 0 ? (
          <Text variant="caption" color="secondary">
            {row.skippedBars}
          </Text>
        ) : (
          <Text variant="caption" color="secondary">
            —
          </Text>
        ),
    },
    {
      title: t('signals:actions'),
      key: 'actions',
      render: (_, row) => (
        <>
          <Button size="small" type="link" onClick={() => onSelect(row.id)}>
            {t('signals:lab.view')}
          </Button>
          {row.status === 'pending' || row.status === 'running' ? (
            <Button size="small" danger type="link" loading={cancelLoading} onClick={() => onCancel(row.id)}>
              {t('common:actions.cancel')}
            </Button>
          ) : null}
        </>
      ),
    },
  ];

  const showSignalMarket = isWatchlistBacktest(selectedJob) || signals.some((s) => Boolean(s.symbol));
  const signalColumns: ColumnsType<ScannerBacktestSignal> = [
    ...(showSignalMarket
      ? [
          {
            title: t('signals:symbol'),
            key: 'market',
            render: (_: unknown, row: ScannerBacktestSignal) => (
              <Text variant="label" mono>
                {row.symbol ? formatSymbolDisplay(row.symbol) : '—'}
              </Text>
            ),
          } satisfies ColumnsType<ScannerBacktestSignal>[number],
        ]
      : []),
    {
      title: t('signals:lab.signalAt'),
      dataIndex: 'signalAt',
      render: (v: string) => (
        <Text variant="caption" mono>
          {formatExactDateTime(v, i18n.language)}
        </Text>
      ),
    },
    {
      title: t('signals:lab.close'),
      dataIndex: 'closePrice',
      render: (v: number) => (
        <Text variant="label" mono>
          {v}
        </Text>
      ),
    },
    {
      title: t('signals:lab.side'),
      key: 'side',
      render: (_, row) => (
        <Text variant="caption">{sideLabel(t, row.trade?.side)}</Text>
      ),
    },
    {
      title: t('signals:lab.outcome'),
      key: 'outcome',
      render: (_, row) => (
        <Text variant="caption" color={row.trade?.outcome === 'tp' ? 'success' : row.trade?.outcome === 'sl' ? 'error' : 'secondary'}>
          {outcomeLabel(t, row.trade?.outcome)}
        </Text>
      ),
    },
    {
      title: t('signals:lab.pnl'),
      key: 'pnl',
      render: (_, row) => (
        <Text variant="caption" color={tradePnlTone(row.trade)} mono>
          {row.trade == null ? '—' : formatChangePercent(row.trade.pnlPct)}
        </Text>
      ),
    },
    {
      title: t('signals:summary'),
      dataIndex: 'summary',
      ellipsis: true,
    },
    {
      title: t('signals:lab.ret1d'),
      dataIndex: 'return1d',
      render: (v: number | undefined) => (
        <Text variant="caption" color={changeTone(v)} mono>
          {v == null ? '—' : formatChangePercent(v)}
        </Text>
      ),
    },
    {
      title: t('signals:lab.ret5d'),
      dataIndex: 'return5d',
      render: (v: number | undefined) => (
        <Text variant="caption" color={changeTone(v)} mono>
          {v == null ? '—' : formatChangePercent(v)}
        </Text>
      ),
    },
    {
      title: t('signals:lab.ret20d'),
      dataIndex: 'return20d',
      render: (v: number | undefined) => (
        <Text variant="caption" color={changeTone(v)} mono>
          {v == null ? '—' : formatChangePercent(v)}
        </Text>
      ),
    },
  ];

  return (
    <Stack>
      <div>
        <Text variant="h4" color="primary">
          {t('signals:lab.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('signals:lab.hint')}
        </Text>
      </div>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:lab.source')}
          </Text>
          <Select
            value={source}
            aria-label={t('signals:lab.source')}
            style={{ minWidth: 160 }}
            options={[
              { value: 'rule', label: t('signals:lab.rule') },
              { value: 'profile', label: t('signals:lab.profile') },
            ]}
            onChange={setSource}
          />
        </Field>
        {source === 'rule' ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.rule')}
            </Text>
            <Select
              value={ruleId || undefined}
              aria-label={t('signals:lab.rule')}
              style={{ minWidth: 240 }}
              placeholder={t('signals:lab.pickRule')}
              options={rules.map((r) => ({
                value: r.id,
                label: `${r.interval} · ${describeRule(r)}`,
              }))}
              onChange={setRuleId}
            />
          </Field>
        ) : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.profile')}
            </Text>
            <Select
              value={profileId}
              aria-label={t('signals:lab.profile')}
              style={{ minWidth: 240 }}
              options={[
                { value: 'built_in', label: t('signals:custom.builtIn') },
                ...profiles.map((p) => ({
                  value: p.id,
                  label: p.name || p.id,
                })),
              ]}
              onChange={setProfileId}
            />
          </Field>
        )}
        {source === 'profile' ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.universe')}
            </Text>
            <Select
              value={scope}
              aria-label={t('signals:lab.universe')}
              style={{ minWidth: 160 }}
              options={[
                { value: 'symbol', label: t('signals:lab.oneCoin') },
                { value: 'watchlist', label: t('signals:lab.watchlist') },
              ]}
              onChange={setScope}
            />
          </Field>
        ) : null}
        {watchlistRun ? null : (
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:exchange')}
          </Text>
          <Select
            value={exchange}
            aria-label={t('signals:exchange')}
            style={{ minWidth: 120 }}
            options={['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'].map((e) => ({ value: e, label: e }))}
            onChange={setExchange}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:symbol')}
          </Text>
          <SymbolSuggest
            exchange={exchange}
            value={symbol}
            onChange={setSymbol}
            aria-label={t('signals:symbol')}
          />
        </Field>
        )}
        {watchlistRun ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.startingBalance')}
            </Text>
            <InputNumber
              min={1}
              value={startingBalance}
              aria-label={t('signals:lab.startingBalance')}
              style={{ minWidth: 140 }}
              onChange={(v) => setStartingBalance(typeof v === 'number' ? v : 10000)}
            />
          </Field>
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.riskPct')}
            </Text>
            <InputNumber
              min={0.1}
              max={25}
              step={0.1}
              value={riskPct}
              aria-label={t('signals:lab.riskPct')}
              style={{ minWidth: 100 }}
              onChange={(v) => setRiskPct(typeof v === 'number' ? v : 1)}
            />
          </Field>
          <Field>
            <Text variant="caption" color="secondary">
              {t('signals:lab.maxOpenRisk')}
            </Text>
            <InputNumber
              min={0.1}
              max={100}
              step={0.5}
              value={maxOpenRiskPct}
              aria-label={t('signals:lab.maxOpenRisk')}
              style={{ minWidth: 100 }}
              onChange={(v) => setMaxOpenRiskPct(typeof v === 'number' ? v : 3)}
            />
          </Field>
        ) : null}
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:lab.range')}
          </Text>
          <Select
            value={rangeKey}
            aria-label={t('signals:lab.range')}
            style={{ minWidth: 120 }}
            options={rangeOptions}
            onChange={setRangeKey}
          />
        </Field>
        <Button
          type="primary"
          loading={startLoading}
          disabled={!canStart}
          onClick={() =>
            onStart({
              ruleId: source === 'rule' ? ruleId : undefined,
              profileId: source === 'profile' ? profileId : undefined,
              scope: source === 'profile' ? scope : 'symbol',
              startingBalance: watchlistRun ? startingBalance : undefined,
              riskPct: watchlistRun ? riskPct : undefined,
              maxOpenRiskPct: watchlistRun ? maxOpenRiskPct : undefined,
              symbol: watchlistRun ? undefined : symbol.trim().toUpperCase(),
              exchange: watchlistRun ? undefined : (exchange as MarketExchange),
              rangeKey,
            })
          }
        >
          {t('signals:lab.start')}
        </Button>
      </FieldRow>
      {startError != null ? (
        <Alert
          type="error"
          showIcon
          message={t('signals:lab.startFailed')}
          description={rtkErrorMessage(startError, { resource: t('signals:resource') })}
        />
      ) : null}
      <DataTableCard>
        <DataTable
          rowKey={(r) => r.id}
          loading={loading}
          dataSource={jobs}
          columns={jobColumns}
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: <Empty description={t('signals:lab.empty')} /> }}
        />
      </DataTableCard>
      <Drawer
        open={Boolean(selectedId)}
        onClose={() => onSelect('')}
        width={880}
        title={
          selectedJob
            ? `${
                isWatchlistBacktest(selectedJob)
                  ? t('signals:lab.watchlistCoins', { count: watchlistMarketCount(selectedJob) })
                  : formatSymbolDisplay(selectedJob.symbol)
              } · ${selectedJob.interval}`
            : t('signals:lab.signals')
        }
      >
        <SignalDrawerBody>
          {selectedJob?.errorMessage ? (
            <Alert type="error" showIcon message={selectedJob.errorMessage} />
          ) : null}
          {performance && ((performance.signalCount ?? 0) > 0 || performance.book) ? (
            <PerformanceSummary perf={performance} />
          ) : null}
          {(performance?.markets?.length ?? 0) > 0 ? (
            <DataTable
              rowKey={(r) => `${r.exchange}-${r.symbol}`}
              size="small"
              pagination={false}
              dataSource={performance?.markets}
              columns={[
                {
                  title: t('signals:symbol'),
                  key: 'sym',
                  render: (_, row) => (
                    <Text variant="label" mono>
                      {formatSymbolDisplay(row.symbol)}
                    </Text>
                  ),
                },
                {
                  title: t('signals:lab.marketStatus'),
                  key: 'st',
                  render: (_, row) => (
                    <Text variant="caption" color={row.status === 'error' || row.status === 'no_data' ? 'error' : 'secondary'}>
                      {marketStatusLabel(t, row)}
                      {row.error ? ` — ${row.error}` : ''}
                    </Text>
                  ),
                },
                {
                  title: t('signals:lab.signals'),
                  dataIndex: 'signalCount',
                },
                {
                  title: t('signals:lab.tradeWinRate'),
                  key: 'wr',
                  render: (_, row) => (
                    <Text variant="caption" mono>
                      {row.trades?.winRate == null ? '—' : `${row.trades.winRate}%`}
                    </Text>
                  ),
                },
                {
                  title: t('signals:lab.avgPnl'),
                  key: 'avg',
                  render: (_, row) => (
                    <Text variant="caption" color={changeTone(row.trades?.average)} mono>
                      {row.trades?.average == null ? '—' : formatChangePercent(row.trades.average)}
                    </Text>
                  ),
                },
              ]}
            />
          ) : null}
          {selectedJob?.kind === 'custom_profile' && (selectedJob.skippedBars ?? 0) > 0 ? (
            <Alert
              type="warning"
              showIcon
              message={t('signals:lab.skippedDetail', { count: selectedJob.skippedBars })}
              description={
                (selectedJob.missingFactors ?? []).length
                  ? t('signals:lab.missingFactors', {
                      factors: (selectedJob.missingFactors ?? [])
                        .map((id) => t(`signals:custom.factors.${id}`, { defaultValue: id }))
                        .join(', '),
                    })
                  : undefined
              }
            />
          ) : null}
          <DataTable
            rowKey={(r) => r.id}
            loading={signalsLoading}
            dataSource={signals}
            columns={signalColumns}
            pagination={{ pageSize: 20 }}
            locale={{ emptyText: <Empty description={t('signals:lab.noSignals')} /> }}
            size="small"
            expandable={{
              rowExpandable: (row) => hasBacktestHitDetails(row),
              expandedRowRender: (row) => (
                <FactorWhy>
                  {row.trade ? (
                    <>
                      <Text variant="caption" color="secondary">
                        {t('signals:lab.hitTrade')}
                      </Text>
                      <TradeDetail trade={row.trade} />
                    </>
                  ) : null}
                  {(row.factors?.length ?? 0) > 0 ? (
                    <>
                      <Text variant="caption" color="secondary">
                        {t('signals:lab.hitFactors')}
                      </Text>
                      {row.factors?.map((f) => (
                        <FactorLine key={f.id}>
                          <Text variant="caption" color="primary">
                            {t(`signals:custom.factors.${f.id}`, { defaultValue: f.label })}
                            {f.status === 'idle' ? ` (${t('signals:custom.idle')})` : ''}
                          </Text>
                          <Text variant="caption" mono color="secondary">
                            {t('signals:lab.factorScore', { score: Math.round(f.score) })}
                          </Text>
                          <Text variant="caption" mono color="secondary">
                            {t('signals:lab.factorWeight', { pct: f.weightPct })}
                          </Text>
                          <Text variant="caption" mono color={f.effect && f.effect < 0 ? 'error' : 'success'}>
                            {f.status === 'used' && f.effect != null
                              ? f.effect > 0
                                ? `+${f.effect}`
                                : String(f.effect)
                              : '—'}
                          </Text>
                        </FactorLine>
                      ))}
                    </>
                  ) : null}
                </FactorWhy>
              ),
            }}
          />
        </SignalDrawerBody>
      </Drawer>
    </Stack>
  );
}
