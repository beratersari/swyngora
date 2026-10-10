import { Fragment, useState } from 'react';
import { Alert, Button, Empty, InputNumber, Select, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import {
  rtkErrorMessage,
  useGetMarketTapeBacktestQuery,
  type TapeBacktestHorizon,
  type TapeBacktestSignal,
  type TapeBacktestVenue,
} from '@/libs/api';
import { changeTone, formatChangePercent, formatExactDateTime, formatPrice } from '@/libs/utils';
import { TAPE_CONDITIONS, TAPE_DEFAULT_SYMBOL, TAPE_DEFAULTS, TAPE_RANGES } from './constants';
import {
  formatTapeReading,
  isTapeCondition,
  tapeDefaults,
  tapeExchanges,
  tapeQueryFromDraft,
  tapeSides,
  tapeThresholdBounds,
} from './helpers';
import { Field, FieldRow, PerfGrid, Stack, VenueBlock } from './TapeBacktestPanel.styles';
import type { TapeCondition, TapeDraft } from './TapeBacktestPanel.types';

const INITIAL: TapeDraft = {
  symbol: TAPE_DEFAULT_SYMBOL,
  condition: 'cvd',
  side: TAPE_DEFAULTS.cvd.side,
  exchange: 'all',
  days: 7,
  threshold: TAPE_DEFAULTS.cvd.threshold,
};

type SubmittedTape = {
  draft: TapeDraft;
  now: number;
};

function horizonById(rows: TapeBacktestHorizon[] | undefined, id: string): TapeBacktestHorizon | undefined {
  return rows?.find((row) => row.horizon === id);
}

function VenueResult({
  venue,
  condition,
  title,
}: {
  venue: TapeBacktestVenue;
  condition: string;
  title: string;
}) {
  const { t } = useTranslation(['signals']);
  const horizons = ['1h', '4h', '24h'].map((id) => horizonById(venue.horizons, id) ?? { horizon: id, sampleCount: 0 });
  const rows: { key: string; label: string; render: (h: TapeBacktestHorizon) => string }[] = [
    {
      key: 'avg',
      label: t('signals:tape.avg'),
      render: (h) => (h.average == null ? '—' : formatChangePercent(h.average)),
    },
    {
      key: 'med',
      label: t('signals:tape.median'),
      render: (h) => (h.median == null ? '—' : formatChangePercent(h.median)),
    },
    {
      key: 'win',
      label: t('signals:tape.win'),
      render: (h) => (h.winRate == null ? '—' : `${h.winRate}%`),
    },
    {
      key: 'drop',
      label: t('signals:tape.worst'),
      render: (h) => (h.worstDrop == null ? '—' : formatChangePercent(h.worstDrop)),
    },
    {
      key: 'n',
      label: t('signals:tape.samples'),
      render: (h) => String(h.sampleCount ?? 0),
    },
  ];
  const columns: ColumnsType<TapeBacktestSignal> = [
    {
      title: t('signals:tape.when'),
      dataIndex: 'at',
      render: (value: string | undefined) => (value ? formatExactDateTime(value) : '—'),
    },
    {
      title: t('signals:tape.value'),
      dataIndex: 'value',
      render: (value: number | undefined) => formatTapeReading(condition, value),
    },
    {
      title: t('signals:tape.close'),
      dataIndex: 'close',
      render: (value: number | undefined) => (value ? formatPrice(value) : '—'),
    },
    {
      title: '1h',
      dataIndex: 'return1h',
      render: (value: number | undefined) => formatChangePercent(value),
    },
    {
      title: '4h',
      dataIndex: 'return4h',
      render: (value: number | undefined) => formatChangePercent(value),
    },
    {
      title: '24h',
      dataIndex: 'return24h',
      render: (value: number | undefined) => formatChangePercent(value),
    },
  ];
  return (
    <VenueBlock>
      <Text variant="label" color="primary">
        {title}
        {venue.signalCount != null ? ` · ${venue.signalCount}` : ''}
        {venue.bars != null ? ` · ${t('signals:tape.bars', { count: venue.bars })}` : ''}
      </Text>
      {venue.error ? <Alert type="warning" showIcon message={venue.error} /> : null}
      {venue.note ? (
        <Text variant="caption" color="secondary">
          {venue.note}
        </Text>
      ) : null}
      <PerfGrid>
        <Text variant="caption" color="secondary">
          {' '}
        </Text>
        {horizons.map((h) => (
          <Text key={h.horizon} variant="caption" color="secondary" mono>
            {h.horizon}
          </Text>
        ))}
        {rows.map((row) => (
          <Fragment key={row.key}>
            <Text variant="caption" color="secondary">
              {row.label}
            </Text>
            {horizons.map((h) => (
              <Text
                key={`${row.key}-${h.horizon}`}
                variant="caption"
                mono
                color={
                  row.key === 'avg' || row.key === 'med' || row.key === 'drop'
                    ? changeTone(row.key === 'med' ? h.median : row.key === 'drop' ? h.worstDrop : h.average)
                    : 'primary'
                }
              >
                {row.render(h)}
              </Text>
            ))}
          </Fragment>
        ))}
      </PerfGrid>
      {(venue.signals?.length ?? 0) === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('signals:tape.empty')} />
      ) : (
        <Table
          size="small"
          pagination={false}
          rowKey={(row) => `${row.at ?? ''}-${row.value ?? ''}`}
          columns={columns}
          dataSource={venue.signals}
          scroll={{ x: 720 }}
        />
      )}
    </VenueBlock>
  );
}

export function TapeBacktestPanel() {
  const { t } = useTranslation(['signals']);
  const [draft, setDraft] = useState<TapeDraft>(INITIAL);
  // Keep `now` from the click. A fresh clock on each render changes `from` and restarts the request.
  const [submitted, setSubmitted] = useState<SubmittedTape>(() => ({ draft: INITIAL, now: Date.now() }));
  const query = useGetMarketTapeBacktestQuery(tapeQueryFromDraft(submitted.draft, submitted.now), {
    skip: submitted.draft.symbol.trim() === '',
  });
  const condition = isTapeCondition(draft.condition) ? draft.condition : 'cvd';
  const bounds = tapeThresholdBounds(condition);
  const suggestExchange = draft.exchange === 'all' || draft.exchange === 'okx' ? 'binance' : draft.exchange;

  function setCondition(next: TapeCondition) {
    const defaults = tapeDefaults(next);
    setDraft((cur) => ({
      ...cur,
      condition: next,
      side: defaults.side,
      threshold: defaults.threshold,
      exchange: tapeExchanges(next).includes(cur.exchange as 'all') ? cur.exchange : 'all',
    }));
  }

  return (
    <Stack>
      <Text variant="h4" color="primary">
        {t('signals:tape.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('signals:tape.hint')}
      </Text>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:symbol')}
          </Text>
          <SymbolSuggest
            exchange={suggestExchange}
            value={draft.symbol}
            onChange={(symbol) => setDraft((cur) => ({ ...cur, symbol }))}
            aria-label={t('signals:symbol')}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:tape.condition')}
          </Text>
          <Select
            aria-label={t('signals:tape.condition')}
            value={condition}
            options={TAPE_CONDITIONS.map((id) => ({
              value: id,
              label: t(`signals:tape.conditions.${id}`),
            }))}
            onChange={(value) => setCondition(value)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:tape.side')}
          </Text>
          <Select
            aria-label={t('signals:tape.side')}
            value={draft.side}
            options={tapeSides(condition).map((id) => ({
              value: id,
              label: t(`signals:tape.sides.${id}`),
            }))}
            onChange={(value) => setDraft((cur) => ({ ...cur, side: value }))}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:exchange')}
          </Text>
          <Select
            aria-label={t('signals:exchange')}
            value={draft.exchange}
            options={tapeExchanges(condition).map((id) => ({
              value: id,
              label: id === 'all' ? t('signals:tape.all') : id,
            }))}
            onChange={(value) => setDraft((cur) => ({ ...cur, exchange: value }))}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:tape.threshold')}
          </Text>
          <InputNumber
            aria-label={t('signals:tape.threshold')}
            value={draft.threshold}
            min={bounds.min}
            max={bounds.max}
            step={bounds.step}
            onChange={(value) => {
              if (typeof value === 'number') setDraft((cur) => ({ ...cur, threshold: value }));
            }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:tape.range')}
          </Text>
          <Select
            aria-label={t('signals:tape.range')}
            value={draft.days}
            options={TAPE_RANGES.map((row) => ({
              value: row.days,
              label: t(`signals:tape.ranges.${row.key}`),
            }))}
            onChange={(value) => setDraft((cur) => ({ ...cur, days: value }))}
          />
        </Field>
        <Button type="primary" onClick={() => setSubmitted({ draft: { ...draft }, now: Date.now() })}>
          {t('signals:tape.run')}
        </Button>
      </FieldRow>
      {query.isError ? (
        <Alert type="error" showIcon message={rtkErrorMessage(query.error, { resource: t('signals:tape.title') })} />
      ) : null}
      {query.isFetching ? (
        <Text variant="caption" color="secondary">
          {t('signals:tape.loading')}
        </Text>
      ) : null}
      {query.data ? (
        <>
          {!query.data.definite ? (
            <Alert
              type="warning"
              showIcon
              message={t('signals:tape.withheld', { names: (query.data.missing ?? []).join(', ') || '—' })}
            />
          ) : (
            <Text variant="caption" color="secondary">
              {t('signals:tape.definite')}
            </Text>
          )}
          {query.data.summary ? (
            <Text variant="body" color="primary">
              {query.data.summary}
            </Text>
          ) : null}
          {query.data.combined ? (
            <VenueResult
              venue={query.data.combined}
              condition={query.data.condition ?? condition}
              title={t('signals:tape.combined')}
            />
          ) : null}
          {(query.data.venues ?? []).map((venue) => (
            <VenueResult
              key={venue.exchange ?? 'venue'}
              venue={venue}
              condition={query.data?.condition ?? condition}
              title={venue.exchange ?? ''}
            />
          ))}
          {query.data.note ? (
            <Text variant="caption" color="secondary">
              {query.data.note}
            </Text>
          ) : null}
        </>
      ) : null}
    </Stack>
  );
}
