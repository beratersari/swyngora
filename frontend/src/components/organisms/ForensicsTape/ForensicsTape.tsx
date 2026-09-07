import { Skeleton, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { ForensicsPhase, ForensicsPhaseId } from '@/libs/types/forensics.types';
import { formatForensicsTime, forensicsPhaseById } from '@/libs/utils/forensics';
import {
  ChangeTable,
  MetricGrid,
  MetricLabel,
  MetricValue,
  PhaseCard,
  PhaseGrid,
  PhaseHead,
  Stack,
} from './ForensicsTape.styles';
import type { ForensicsTapeProps } from './ForensicsTape.types';

const PHASES: ForensicsPhaseId[] = ['before', 'during', 'after'];

function dash(v: string | undefined): string {
  return v && v.trim() ? v : '—';
}

function optionsTapeValue(phase?: ForensicsPhase): string {
  const o = phase?.options;
  if (!o?.complete) return '—';
  const when =
    o.started === 'before'
      ? o.lead
        ? `${o.lead} before the move`
        : 'before the move'
      : o.started === 'during'
        ? 'during the move'
        : '';
  const cluster =
    o.together && o.together >= 3 ? '3 series together' : o.together === 2 ? '2 series together' : '';
  const follow =
    o.after === 'continued'
      ? 'kept going'
      : o.after === 'reversed'
        ? 'turned the other way'
        : o.after === 'faded'
          ? 'went quiet'
          : o.after === 'mixed'
            ? 'mixed after'
            : '';
  const bits = [
    o.unusual ? (when ? `unusual ${when}` : 'unusual') : '',
    cluster,
    follow,
    [o.oiChangePct, o.oiDirection].filter(Boolean).join(' '),
    o.leanTo ? `lean ${o.leanTo}` : '',
    o.dvolDelta ? `IV ${o.dvolDelta}` : '',
  ].filter((s) => s && s.trim());
  return bits[0] ? bits.slice(0, 4).join(' · ') : '—';
}

function PhaseMetrics({ phase }: { phase?: ForensicsPhase }) {
  const { t } = useTranslation('forensics');
  const rows: { label: string; value: string }[] = [
    { label: t('price'), value: dash(phase?.price?.close) },
    {
      label: t('change'),
      value: [phase?.price?.change, phase?.price?.changePct].filter(Boolean).join(' ') || '—',
    },
    {
      label: t('range'),
      value: [phase?.price?.range, phase?.price?.rangePct].filter(Boolean).join(' ') || '—',
    },
    { label: t('volume'), value: dash(phase?.flow?.volume) },
    { label: t('buyShare'), value: dash(phase?.flow?.buyShare) },
    { label: t('vwap'), value: dash(phase?.flow?.vwap) },
    {
      label: t('vsTypical'),
      value: phase?.flow?.typicalKnown
        ? [phase.flow.volumeRatio, phase.flow.volumeGrade].filter(Boolean).join(' ') || '—'
        : '—',
    },
    { label: t('poc'), value: dash(phase?.profile?.poc) },
    { label: t('book'), value: dash(phase?.book?.summary ?? phase?.book?.midDelta) },
    {
      label: t('oi'),
      value: [phase?.futures?.oiChange, phase?.futures?.oiChangePct].filter(Boolean).join(' ') || '—',
    },
    {
      label: t('funding'),
      value: [phase?.futures?.fundingFrom, phase?.futures?.fundingTo].filter(Boolean).join(' → ') || '—',
    },
    {
      label: t('liquidations'),
      value:
        phase?.futures?.longLiq || phase?.futures?.shortLiq
          ? `L ${dash(phase.futures.longLiq)} / S ${dash(phase.futures.shortLiq)}`
          : '—',
    },
    {
      label: t('options'),
      value: optionsTapeValue(phase),
    },
  ];
  return (
    <MetricGrid>
      {rows.map((row) => (
        <span key={row.label} style={{ display: 'contents' }}>
          <MetricLabel>
            <Text variant="caption" color="secondary">
              {row.label}
            </Text>
          </MetricLabel>
          <MetricValue>
            <Text variant="bodySm">{row.value}</Text>
          </MetricValue>
        </span>
      ))}
    </MetricGrid>
  );
}

export function ForensicsTape({ venue, isLoading }: ForensicsTapeProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 8 }} />;
  }
  if (!venue) return null;
  return (
    <Stack>
      {venue.summary ? (
        <Text variant="bodySm" color="secondary">
          {venue.summary}
        </Text>
      ) : null}
      {venue.error ? (
        <Text variant="bodySm" color="secondary">
          {venue.error}
        </Text>
      ) : null}
      <PhaseGrid>
        {PHASES.map((id) => {
          const phase = forensicsPhaseById(venue, id);
          return (
            <PhaseCard key={id} $incomplete={phase ? !phase.complete : true}>
              <PhaseHead>
                <Text variant="bodySm">{t(`phases.${id}`)}</Text>
                <Text variant="caption" color="secondary">
                  {formatForensicsTime(phase?.from)} → {formatForensicsTime(phase?.to)}
                </Text>
                {!phase?.complete ? (
                  <Text variant="caption" color="secondary">
                    {t('incomplete')}
                  </Text>
                ) : null}
              </PhaseHead>
              <PhaseMetrics phase={phase} />
              {phase?.summary ? (
                <Text variant="caption" color="secondary">
                  {phase.summary}
                </Text>
              ) : null}
            </PhaseCard>
          );
        })}
      </PhaseGrid>
      {venue.changes && venue.changes.length > 0 ? (
        <ChangeTable>
          <Text variant="bodySm">{t('changes')}</Text>
          <Table
            size="small"
            pagination={false}
            rowKey={(row) => `${row.metric}-${row.path}`}
            dataSource={venue.changes}
            columns={[
              { title: t('metric'), dataIndex: 'metric', key: 'metric' },
              { title: t('phases.before'), dataIndex: 'before', key: 'before' },
              { title: t('phases.during'), dataIndex: 'during', key: 'during' },
              { title: t('phases.after'), dataIndex: 'after', key: 'after' },
              { title: t('path'), dataIndex: 'path', key: 'path' },
            ]}
          />
        </ChangeTable>
      ) : null}
      {venue.events && venue.events.length > 0 ? (
        <div>
          <Text variant="bodySm">{t('events')}</Text>
          {venue.events.map((ev, i) => (
            <Text key={`${ev.at ?? i}-${ev.title ?? ''}`} variant="caption" color="secondary">
              {formatForensicsTime(ev.at)} · {ev.title ?? ev.kind} — {ev.summary}
            </Text>
          ))}
        </div>
      ) : null}
    </Stack>
  );
}
