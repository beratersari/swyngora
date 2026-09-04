import { useMemo, useState } from 'react';
import { Select } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { ForensicsSimilar } from '@/components/organisms/ForensicsSimilar';
import { ForensicsTape } from '@/components/organisms/ForensicsTape';
import { rtkErrorMessage, useGetMarketForensicsQuery } from '@/libs/api';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import {
  FORENSICS_DURINGS,
  FORENSICS_EXCHANGES,
  FORENSICS_WINDOWS,
} from './ForensicsPage.constants';
import {
  datetimeLocalToUtc,
  parseForensicsAt,
  parseForensicsDuring,
  parseForensicsExchange,
  parseForensicsSymbol,
  parseForensicsWindow,
  utcToDatetimeLocal,
} from './ForensicsPage.helpers';
import { forensicsVenueOptions, pickForensicsVenue } from '@/libs/utils/forensics';
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
  const symbol = parseForensicsSymbol(searchParams.get('symbol'));
  const exchange = parseForensicsExchange(searchParams.get('exchange'));
  const windowId = parseForensicsWindow(searchParams.get('window'));
  const during = parseForensicsDuring(searchParams.get('during'));
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
    { skip: !symbol || !at, refetchOnFocus: true },
  );
  const report = rtkCurrent(query) as ForensicsReport | undefined;
  const loading = rtkCurrentPending(query);
  const venues = useMemo(() => forensicsVenueOptions(report?.tape), [report?.tape]);
  const venue = pickForensicsVenue(report?.tape, venueId);

  return (
    <PageStack>
      <PageHeader
        eyebrow={t('forensics:eyebrow')}
        title={t('forensics:title')}
        subtitle={t('forensics:subtitle')}
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
        {venues.length > 1 ? (
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

      {query.isError ? (
        <ErrorHint>
          <Text variant="bodySm">{t('forensics:loadFailed')}</Text>
          <Text variant="caption" color="secondary">
            {rtkErrorMessage(query.error, { resource: t('forensics:resource') })}
          </Text>
        </ErrorHint>
      ) : null}

      {!symbol || !at ? (
        <EmptyHint>
          <Text variant="bodySm" color="secondary">
            {t('forensics:emptyHint')}
          </Text>
        </EmptyHint>
      ) : null}

      {report?.summary ? (
        <SummaryBlock>
          <Text variant="bodySm">{report.summary}</Text>
          <Text variant="caption" color="secondary">
            {report.note || t('forensics:disclaimer')}
          </Text>
        </SummaryBlock>
      ) : null}

      <ForensicsTape venue={venue} isLoading={loading} />
      <ForensicsSimilar
        similar={report?.similar}
        similarError={report?.similarError}
        isLoading={loading}
      />
    </PageStack>
  );
}
