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
import { Field, FieldRow, SignalDrawerBody, Stack } from './SignalsBacktestPanel.styles';
import type { SignalsBacktestPanelProps } from './SignalsBacktestPanel.types';

function statusVariant(status: string): 'live' | 'paused' | 'status' | 'delist' {
  if (status === 'completed') return 'live';
  if (status === 'running' || status === 'pending') return 'status';
  if (status === 'failed') return 'delist';
  return 'paused';
}

export function SignalsBacktestPanel({
  rules,
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
  const [ruleId, setRuleId] = useState(rules[0]?.id ?? '');
  const [exchange, setExchange] = useState<string>('binance');
  const [symbol, setSymbol] = useState('BTCUSDT');
  const [rangeKey, setRangeKey] = useState(rangeOptions[1]?.value ?? rangeOptions[0]?.value ?? '90d');

  const ruleById = useMemo(() => new Map(rules.map((r) => [r.id, r])), [rules]);

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
      title: t('signals:rules.type'),
      key: 'rule',
      render: (_, row) => {
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
          disabled={!ruleId || !symbol.trim() || startLoading}
          onClick={() =>
            onStart({
              ruleId,
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
          <DataTable
            rowKey={(r) => r.id}
            loading={signalsLoading}
            dataSource={signals}
            columns={signalColumns}
            pagination={{ pageSize: 20 }}
            locale={{ emptyText: <Empty description={t('signals:lab.noSignals')} /> }}
            size="small"
          />
        </SignalDrawerBody>
      </Drawer>
    </Stack>
  );
}
