import { useState } from 'react';
import { Button, DatePicker, InputNumber, Segmented } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { STRESS_START_MOVES } from './constants';
import { money, portfolioChange, replayMoveLine, replayPreset, utcDay } from './helpers';
import type { PortfolioStressPanelProps, StressMode, StressMove } from './PortfolioStressPanel.types';
import type { TFunction } from 'i18next';

function grainLabel(t: TFunction<'portfolio'>, interval?: string): string {
  if (interval === '1h') return t('stress.grain.1h');
  if (interval === '1d') return t('stress.grain.1d');
  return interval ?? '';
}

function gapReason(t: TFunction<'portfolio'>, reason?: string): string {
  switch (reason) {
    case 'no_start':
      return t('stress.gaps.no_start');
    case 'no_end':
      return t('stress.gaps.no_end');
    case 'same_close':
      return t('stress.gaps.same_close');
    case 'out_of_range':
      return t('stress.gaps.out_of_range');
    case 'no_history':
      return t('stress.gaps.no_history');
    case 'fetch_failed':
      return t('stress.gaps.fetch_failed');
    default:
      return reason ?? '';
  }
}

export function PortfolioStressPanel({ disabled, isLoading, report, errorMessage, onPreview }: PortfolioStressPanelProps) {
  const { t } = useTranslation('portfolio');
  const [mode, setMode] = useState<StressMode>('moves');
  const [moves, setMoves] = useState<StressMove[]>(STRESS_START_MOVES);
  const initial = replayPreset('7');
  const [from, setFrom] = useState(initial.from);
  const [to, setTo] = useState(initial.to);
  const [localError, setLocalError] = useState<string | null>(null);
  const shownError = errorMessage || localError;
  const moved = new Set((report?.replay?.moves ?? []).map((row) => row.asset).filter((asset): asset is string => Boolean(asset)));

  function applyPreset(kind: 'yesterday' | '7' | '30') {
    const next = replayPreset(kind);
    setFrom(next.from);
    setTo(next.to);
    setLocalError(null);
  }

  return (
    <section data-testid="portfolio-stress">
      <Text variant="h4" color="primary">
        {t('stress.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('stress.hint')}
      </Text>
      <Segmented
        style={{ marginTop: 8 }}
        value={mode}
        onChange={(value) => setMode(value as StressMode)}
        options={[
          { label: t('stress.modeMoves'), value: 'moves' },
          { label: t('stress.modePeriod'), value: 'period' },
        ]}
      />
      {mode === 'moves' ? (
        <>
          {moves.map((row, i) => (
            <div key={row.asset} style={{ display: 'flex', gap: 8, marginTop: 8, alignItems: 'center' }}>
              <Text variant="body" color="primary">
                {row.asset}
              </Text>
              <InputNumber
                aria-label={t('stress.move', { asset: row.asset })}
                value={row.movePct}
                step={1}
                onChange={(value) => {
                  const next = [...moves];
                  next[i] = { ...row, movePct: typeof value === 'number' ? value : row.movePct };
                  setMoves(next);
                }}
              />
            </div>
          ))}
          <Button
            style={{ marginTop: 8 }}
            type="primary"
            disabled={disabled}
            loading={isLoading}
            onClick={() => {
              setLocalError(null);
              onPreview({ moves });
            }}
          >
            {t('stress.preview')}
          </Button>
        </>
      ) : (
        <div data-testid="portfolio-stress-period">
          <Text variant="caption" color="secondary">
            {t('stress.periodHint')}
          </Text>
          <div style={{ display: 'flex', gap: 8, marginTop: 8, flexWrap: 'wrap' }}>
            <Button size="small" onClick={() => applyPreset('yesterday')}>
              {t('stress.yesterday')}
            </Button>
            <Button size="small" onClick={() => applyPreset('7')}>
              {t('stress.last7')}
            </Button>
            <Button size="small" onClick={() => applyPreset('30')}>
              {t('stress.last30')}
            </Button>
          </div>
          <div style={{ display: 'flex', gap: 12, marginTop: 8, flexWrap: 'wrap' }}>
            <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <Text variant="body" color="primary">
                {t('stress.from')}
              </Text>
              <DatePicker
                aria-label={t('stress.from')}
                value={from ? dayjs(from) : null}
                format="YYYY-MM-DD"
                allowClear={false}
                disabledDate={(current) => !!current && current.format('YYYY-MM-DD') > utcDay(-1)}
                onChange={(value) => setFrom(value ? value.format('YYYY-MM-DD') : '')}
              />
            </label>
            <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <Text variant="body" color="primary">
                {t('stress.to')}
              </Text>
              <DatePicker
                aria-label={t('stress.to')}
                value={to ? dayjs(to) : null}
                format="YYYY-MM-DD"
                allowClear={false}
                disabledDate={(current) => !!current && current.format('YYYY-MM-DD') > utcDay(-1)}
                onChange={(value) => setTo(value ? value.format('YYYY-MM-DD') : '')}
              />
            </label>
          </div>
          <Button
            style={{ marginTop: 8 }}
            type="primary"
            disabled={disabled}
            loading={isLoading}
            onClick={() => {
              if (!from || !to) {
                setLocalError(t('stress.needDates'));
                return;
              }
              if (from > to) {
                setLocalError(t('stress.order'));
                return;
              }
              setLocalError(null);
              onPreview({ from, to });
            }}
          >
            {t('stress.replay')}
          </Button>
        </div>
      )}
      {shownError ? (
        <Text variant="caption" color="secondary">
          {shownError}
        </Text>
      ) : null}
      {report ? (
        <div data-testid="portfolio-stress-result">
          {report.replay ? (
            <div data-testid="portfolio-stress-replay">
              <Text variant="caption" color="secondary">
                {grainLabel(t, report.replay.interval)}
              </Text>
              {(report.replay.moves ?? []).map((row) => (
                <Text key={row.asset} variant="body" color="primary">
                  {replayMoveLine(row)}
                </Text>
              ))}
              {(report.replay.missing ?? []).map((gap) => {
                const reason = gapReason(t, gap.reason);
                const symbol = gap.symbol || gap.asset || '';
                const blended = Boolean(gap.asset && gap.symbol && moved.has(gap.asset));
                return (
                  <Text key={`${gap.exchange}-${symbol}-${gap.reason}`} variant="caption" color="secondary">
                    {blended
                      ? t('stress.gapBlend', { symbol, exchange: gap.exchange ?? '', asset: gap.asset ?? '', reason })
                      : t('stress.gapHeld', { symbol, reason })}
                  </Text>
                );
              })}
            </div>
          ) : null}
          <Text variant="body" color="primary">
            {t('stress.equity', {
              now: money(report.equity),
              next: money(report.shockedEquity),
              change: portfolioChange(report.equity, report.equityDelta),
            })}
          </Text>
          <Text variant="body" color="primary">
            {t('stress.pnl', { now: money(report.totalPnl, true), next: money(report.shockedTotalPnl, true) })}
          </Text>
          <Text variant="body" color="primary">
            {t('stress.margin', { now: money(report.marginEquity), next: money(report.shockedMarginEquity) })}
          </Text>
          {(report.liquidated ?? 0) > 0 ? (
            <Text variant="caption" color="secondary">
              {t('stress.liquidated', { n: report.liquidated })}
            </Text>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
