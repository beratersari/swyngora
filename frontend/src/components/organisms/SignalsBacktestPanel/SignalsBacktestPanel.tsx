import { useMemo, useState } from 'react';
import { Alert, Button, Drawer, Empty, Progress, Select } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import {
  rtkErrorMessage,
  type MarketExchange,
  type ScannerBacktest,
  type ScannerBacktestSignal,
} from '@/libs/api';
import {
  changeTone,
  describeRule,
  formatChangePercent,
  formatExactDateTime,
  formatSymbolDisplay,
} from '@/libs/utils';
import { DataTable, DataTableCard } from '@/styles/shared/dataTable.styles';
import { FactorLine, FactorWhy, Field, FieldRow, SignalDrawerBody, Stack } from './SignalsBacktestPanel.styles';
import type { SignalsBacktestPanelProps } from './SignalsBacktestPanel.types';

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
  const [rangeKey, setRangeKey] = useState(rangeOptions[1]?.value ?? rangeOptions[0]?.value ?? '90d');

  const ruleById = useMemo(() => new Map(rules.map((r) => [r.id, r])), [rules]);
  const canStart =
    Boolean(symbol.trim()) &&
    !startLoading &&
    (source === 'rule' ? Boolean(ruleId) : Boolean(profileId));

  const jobColumns: ColumnsType<ScannerBacktest> = [
    {
      title: t('signals:symbol'),
      key: 'symbol',
      render: (_, row) => (
        <Text variant="label" mono>
          {formatSymbolDisplay(row.symbol)}
        </Text>
      ),
    },
    {
      title: t('signals:exchange'),
      dataIndex: 'exchange',
      render: (ex: string) => <BrandTag variant="exchange">{ex}</BrandTag>,
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

  const signalColumns: ColumnsType<ScannerBacktestSignal> = [
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
              symbol: symbol.trim().toUpperCase(),
              exchange: exchange as MarketExchange,
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
        width={720}
        title={
          selectedJob
            ? `${formatSymbolDisplay(selectedJob.symbol)} · ${selectedJob.interval}`
            : t('signals:lab.signals')
        }
      >
        <SignalDrawerBody>
          {selectedJob?.errorMessage ? (
            <Alert type="error" showIcon message={selectedJob.errorMessage} />
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
              rowExpandable: (row) => (row.factors?.length ?? 0) > 0,
              expandedRowRender: (row) => (
                <FactorWhy>
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
                </FactorWhy>
              ),
            }}
          />
        </SignalDrawerBody>
      </Drawer>
    </Stack>
  );
}
