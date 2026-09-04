import { useMemo, useState } from 'react';
import { Segmented, Select } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { ForensicsNowScan } from '@/components/organisms/ForensicsNowScan';
import { ForensicsSimilar } from '@/components/organisms/ForensicsSimilar';
import { ForensicsTape } from '@/components/organisms/ForensicsTape';
import { ForensicsWhy } from '@/components/organisms/ForensicsWhy';
import {
  rtkErrorMessage,
  useGetMarketForensicsQuery,
  useScanMarketForensicsQuery,
} from '@/libs/api';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import {
  FORENSICS_DURINGS,
  FORENSICS_EXCHANGES,
  FORENSICS_LOOKBACKS,
  FORENSICS_WINDOWS,
} from './ForensicsPage.constants';
import {
  datetimeLocalToUtc,
  parseForensicsAt,
  parseForensicsDuring,
  parseForensicsExchange,
  parseForensicsLookback,
  parseForensicsSymbol,
  parseForensicsView,
  parseForensicsWindow,
  utcToDatetimeLocal,
} from './ForensicsPage.helpers';
import { forensicsVenueOptions, pickForensicsVenue } from '@/libs/utils/forensics';
import type { ForensicsScan } from '@/libs/types/forensics.types';
import {
  DatetimeInput,
  EmptyHint,
  ErrorHint,
  Field,
  PageStack,
  SummaryBlock,
  Toolbar,
} from './ForensicsPage.styles';
import type { ForensicsReport } from './ForensicsPage.types';

export function ForensicsPage() {
  const { t } = useTranslation(['forensics', 'common']);
  const [searchParams, setSearchParams] = useSearchParams();
  const view = parseForensicsView(searchParams.get('view'));
  const symbol = parseForensicsSymbol(searchParams.get('symbol'));
  const exchange = parseForensicsExchange(searchParams.get('exchange'));
  const windowId = parseForensicsWindow(searchParams.get('window'));
  const during = parseForensicsDuring(searchParams.get('during'));
  const lookback = parseForensicsLookback(searchParams.get('lookback'));
  const at = parseForensicsAt(searchParams.get('at'));
  const [venueId, setVenueId] = useState('combined');

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

  const query = useGetMarketForensicsQuery(
    { symbol, at, exchange, window: windowId, during },
    { skip: view !== 'event' || !symbol || !at, refetchOnFocus: true },
  );
  const scanQuery = useScanMarketForensicsQuery(
    { exchange, lookback, window: windowId, limit: 12 },
    { skip: view !== 'now', refetchOnFocus: true },
  );
  const report = rtkCurrent(query) as ForensicsReport | undefined;
  const scan = rtkCurrent(scanQuery) as ForensicsScan | undefined;
  const loading = view === 'now' ? rtkCurrentPending(scanQuery) : rtkCurrentPending(query);
  const venues = useMemo(() => forensicsVenueOptions(report?.tape), [report?.tape]);
  const venue = pickForensicsVenue(report?.tape, venueId);

  return (
    <PageStack>
      <PageHeader
        eyebrow={t('forensics:eyebrow')}
        title={t('forensics:title')}
        subtitle={t('forensics:subtitle')}
      />

      <Segmented
        value={view}
        options={[
          { label: t('forensics:view.event'), value: 'event' },
          { label: t('forensics:view.now'), value: 'now' },
        ]}
        onChange={(v) => patchParams({ view: String(v) })}
      />

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary">
            {t('forensics:exchange')}
          </Text>
          <Select
            value={exchange}
            style={{ minWidth: 120 }}
            options={FORENSICS_EXCHANGES.map((e) => ({
              value: e,
              label: e === 'all' ? e : t(`common:exchanges.${e}`, { defaultValue: e }),
            }))}
            onChange={(v) => patchParams({ exchange: v })}
          />
        </Field>
        {view === 'event' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('forensics:symbol')}
              </Text>
              <SymbolSuggest
                exchange={exchange === 'all' ? 'binance' : exchange}
                value={symbol}
                onChange={(v) => patchParams({ symbol: v.trim().toUpperCase() })}
                aria-label={t('forensics:symbol')}
                placeholder="BTCUSDT"
                style={{ minWidth: 160 }}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('forensics:at')}
              </Text>
              <DatetimeInput
                type="datetime-local"
                aria-label={t('forensics:at')}
                value={utcToDatetimeLocal(at)}
                onChange={(e) => {
                  const next = datetimeLocalToUtc(e.target.value);
                  if (next) patchParams({ at: next });
                }}
              />
            </Field>
          </>
        ) : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('forensics:scan.lookback')}
            </Text>
            <Select
              value={lookback}
              style={{ minWidth: 100 }}
              options={FORENSICS_LOOKBACKS.map((w) => ({ value: w, label: w }))}
              onChange={(v) => patchParams({ lookback: v })}
            />
          </Field>
        )}
        <Field>
          <Text variant="caption" color="secondary">
            {t('forensics:window')}
          </Text>
          <Select
            value={windowId}
            style={{ minWidth: 100 }}
            options={FORENSICS_WINDOWS.map((w) => ({ value: w, label: w }))}
            onChange={(v) => patchParams({ window: v })}
          />
        </Field>
        {view === 'event' ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('forensics:during')}
            </Text>
            <Select
              value={during}
              style={{ minWidth: 100 }}
              options={FORENSICS_DURINGS.map((w) => ({ value: w, label: w }))}
              onChange={(v) => patchParams({ during: v })}
            />
          </Field>
        ) : null}
        {view === 'event' && venues.length > 1 ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('forensics:venue')}
            </Text>
            <Select
              value={venues.some((v) => v.value === venueId) ? venueId : venues[0]?.value}
              style={{ minWidth: 120 }}
              options={venues.map((v) => ({
                value: v.value,
                label: v.value === 'combined' ? t('forensics:venueCombined') : v.label,
              }))}
              onChange={setVenueId}
            />
          </Field>
        ) : null}
      </Toolbar>

      {view === 'event' && query.isError ? (
        <ErrorHint>
          <Text variant="bodySm">{t('forensics:loadFailed')}</Text>
          <Text variant="caption" color="secondary">
            {rtkErrorMessage(query.error, { resource: t('forensics:resource') })}
          </Text>
        </ErrorHint>
      ) : null}

      {view === 'now' && scanQuery.isError ? (
        <ErrorHint>
          <Text variant="bodySm">{t('forensics:scan.loadFailed')}</Text>
          <Text variant="caption" color="secondary">
            {rtkErrorMessage(scanQuery.error, { resource: t('forensics:scan.resource') })}
          </Text>
        </ErrorHint>
      ) : null}

      {view === 'event' && (!symbol || !at) ? (
        <EmptyHint>
          <Text variant="bodySm" color="secondary">
            {t('forensics:emptyHint')}
          </Text>
        </EmptyHint>
      ) : null}

      {view === 'now' ? (
        <ForensicsNowScan
          scan={scan}
          isLoading={loading}
          onOpenSymbol={(sym) => patchParams({ view: 'event', symbol: sym })}
        />
      ) : (
        <>
          <ForensicsWhy why={report?.why} isLoading={loading} />
          {report?.note && !report.why ? (
            <SummaryBlock>
              <Text variant="caption" color="secondary">
                {report.note}
              </Text>
            </SummaryBlock>
          ) : null}
          <ForensicsTape venue={venue} isLoading={loading} />
          <ForensicsSimilar
            similar={report?.similar}
            similarError={report?.similarError}
            isLoading={loading}
          />
        </>
      )}
    </PageStack>
  );
}
